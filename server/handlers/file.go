package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"haven/server/models"
	"haven/server/ws"
	"haven/shared"

	"gorm.io/gorm"
)

// FileToken represents a single-use upload or download token.
type FileToken struct {
	Token     string
	UserID    string
	PubKeyHex string
	IsUpload  bool
	FileID    string // for download tokens
	Thumbnail bool   // for download tokens
	Name      string // for upload tokens
	Size      int64  // for upload tokens
	MimeType  string // for upload tokens
	ChannelID *string
	ExpiresAt time.Time
}

// FileTokenStore is a concurrent-safe in-memory token store.
type FileTokenStore struct {
	mu     sync.Mutex
	tokens map[string]*FileToken
}

// NewFileTokenStore creates a new token store.
func NewFileTokenStore() *FileTokenStore {
	s := &FileTokenStore{
		tokens: make(map[string]*FileToken),
	}
	go s.cleanupLoop()
	return s
}

// Put stores a token.
func (s *FileTokenStore) Put(ft *FileToken) {
	s.mu.Lock()
	s.tokens[ft.Token] = ft
	s.mu.Unlock()
}

// Take retrieves and removes a token (single-use).
func (s *FileTokenStore) Take(token string) *FileToken {
	s.mu.Lock()
	defer s.mu.Unlock()
	ft, ok := s.tokens[token]
	if !ok {
		return nil
	}
	delete(s.tokens, token)
	if time.Now().After(ft.ExpiresAt) {
		return nil
	}
	return ft
}

func (s *FileTokenStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, ft := range s.tokens {
			if now.After(ft.ExpiresAt) {
				delete(s.tokens, k)
			}
		}
		s.mu.Unlock()
	}
}

func RegisterFileHandlers(d *Deps, router *ws.Router) {
	router.Register(shared.TypeFileUploadRequest, handleFileUploadRequest(d))
	router.Register(shared.TypeFileDownloadRequest, handleFileDownloadRequest(d))
}

func handleFileUploadRequest(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			Name      string  `json:"name"`
			Size      int64   `json:"size"`
			MimeType  string  `json:"mime_type"`
			ChannelID *string `json:"channel_id"`
		}
		if !parsePayload(msg, &req) || req.Name == "" || req.Size <= 0 || req.MimeType == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "name, size, and mime_type are required")
			return
		}

		// Check attach files permission for channel uploads
		if req.ChannelID != nil && *req.ChannelID != "" {
			if !checkPerm(d, client, msg.Type, msg.ID, shared.PermAttachFiles) {
				return
			}
			if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, *req.ChannelID) {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "no access to channel")
				return
			}
		}

		// Check file size limit
		var srv models.Server
		if err := d.DB.First(&srv).Error; err == nil {
			if req.Size > srv.MaxFileSize {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrFileTooLarge, "file exceeds maximum size")
				return
			}
		}

		token := generateFileToken()
		ft := &FileToken{
			Token:     token,
			UserID:    client.UserID,
			PubKeyHex: client.PubKeyHex,
			IsUpload:  true,
			Name:      req.Name,
			Size:      req.Size,
			MimeType:  req.MimeType,
			ChannelID: req.ChannelID,
			ExpiresAt: time.Now().Add(60 * time.Second),
		}
		d.FileTokens.Put(ft)

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"token":      token,
			"url":        "/upload",
			"expires_in": 60,
		})
	}
}

func handleFileDownloadRequest(d *Deps) ws.HandlerFunc {
	return func(client *ws.Client, msg *ws.WSMessage) {
		var req struct {
			FileID    string `json:"file_id"`
			Thumbnail bool   `json:"thumbnail"`
		}
		if !parsePayload(msg, &req) || req.FileID == "" {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrBadRequest, "file_id is required")
			return
		}

		// Verify file exists and user has access
		var file models.File
		if err := d.DB.First(&file, "id = ?", req.FileID).Error; err != nil {
			ws.SendError(client, msg.Type, msg.ID, shared.ErrNotFound, "file not found")
			return
		}

		// If file belongs to a channel, verify access
		if file.ChannelID != nil && *file.ChannelID != "" {
			if !hasChannelAccess(d.DB, d.Hot, client.UserID, client.PubKey, *file.ChannelID) {
				ws.SendError(client, msg.Type, msg.ID, shared.ErrPermissionDenied, "no access to file")
				return
			}
		}

		token := generateFileToken()
		urlPath := fmt.Sprintf("/files/%s", req.FileID)
		if req.Thumbnail {
			urlPath += "/thumb"
		}

		ft := &FileToken{
			Token:     token,
			UserID:    client.UserID,
			PubKeyHex: client.PubKeyHex,
			IsUpload:  false,
			FileID:    req.FileID,
			Thumbnail: req.Thumbnail,
			ExpiresAt: time.Now().Add(60 * time.Second),
		}
		d.FileTokens.Put(ft)

		ws.SendOK(client, msg.Type, msg.ID, map[string]any{
			"token":      token,
			"url":        urlPath,
			"expires_in": 60,
		})
	}
}

// HandleUpload is the HTTP handler for POST /upload.
func HandleUpload(db *gorm.DB, hub *ws.Hub, tokens *FileTokenStore, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := extractToken(r)
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		ft := tokens.Take(token)
		if ft == nil || !ft.IsUpload {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Parse multipart form (limit to file size + some overhead)
		if err := r.ParseMultipartForm(ft.Size + 1024*1024); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileID := newULID()
		uploadsDir := filepath.Join(dataDir, "uploads")
		os.MkdirAll(uploadsDir, 0700)

		storagePath := filepath.Join(uploadsDir, fileID)
		out, err := os.Create(storagePath)
		if err != nil {
			slog.Error("create upload file", "error", err)
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		written, err := io.Copy(out, file)
		out.Close()
		if err != nil {
			os.Remove(storagePath)
			slog.Error("write upload file", "error", err)
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		dbFile := models.File{
			ID:          fileID,
			UploaderID:  ft.UserID,
			ChannelID:   ft.ChannelID,
			Name:        ft.Name,
			MimeType:    ft.MimeType,
			Size:        written,
			StoragePath: storagePath,
		}

		// Generate thumbnail for images
		hasThumbnail := false
		if strings.HasPrefix(ft.MimeType, "image/") {
			thumbPath := storagePath + ".thumb"
			if generateThumbnail(storagePath, thumbPath) {
				dbFile.ThumbPath = &thumbPath
				hasThumbnail = true
			}
		}

		if err := db.Create(&dbFile).Error; err != nil {
			os.Remove(storagePath)
			slog.Error("create file record", "error", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)

		// Push upload complete event via WS
		eventBytes, _ := ws.MarshalEvent(shared.TypeEventFileUploadComplete, map[string]any{
			"file_id":       fileID,
			"name":          ft.Name,
			"mime_type":     ft.MimeType,
			"size":          written,
			"has_thumbnail": hasThumbnail,
		})
		hub.SendTo(ft.PubKeyHex, eventBytes)
	}
}

// HandleDownload is the HTTP handler for GET /files/{file_id} and /files/{file_id}/thumb.
func HandleDownload(db *gorm.DB, tokens *FileTokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := extractToken(r)
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		ft := tokens.Take(token)
		if ft == nil || ft.IsUpload {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		var file models.File
		if err := db.First(&file, "id = ?", ft.FileID).Error; err != nil {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}

		servePath := file.StoragePath
		if ft.Thumbnail {
			if file.ThumbPath == nil {
				http.Error(w, "no thumbnail available", http.StatusNotFound)
				return
			}
			servePath = *file.ThumbPath
		}

		w.Header().Set("Content-Type", file.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, file.Name))
		http.ServeFile(w, r, servePath)
	}
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func generateFileToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateThumbnail is a placeholder — actual image processing would use an imaging library.
// For now it returns false, meaning no thumbnail is generated.
func generateThumbnail(_, _ string) bool {
	return false
}
