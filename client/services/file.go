package services

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"haven/client/connection"
	"haven/shared"
)

// UploadProgress tracks file upload progress.
type UploadProgress struct {
	FileID    string  `json:"fileId"`
	FileName  string  `json:"fileName"`
	BytesSent int64   `json:"bytesSent"`
	TotalSize int64   `json:"totalSize"`
	Percent   float64 `json:"percent"`
	Done      bool    `json:"done"`
	Error     string  `json:"error,omitempty"`
}

// FileInfo is the frontend-facing file metadata.
type FileInfo struct {
	ID        string `json:"id"`
	FileName  string `json:"fileName"`
	MimeType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// FileService manages file uploads and downloads via tokenized HTTP.
type FileService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
}

// NewFileService creates a new FileService.
func NewFileService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *FileService {
	return &FileService{
		db:      db,
		manager: manager,
		privKey: privKey,
	}
}

// SetContext is called by Wails during startup.
func (f *FileService) SetContext(ctx context.Context) {
	f.ctx = ctx
}

// PickFile opens a native file dialog and returns the selected file path.
func (f *FileService) PickFile() (string, error) {
	return wailsRuntime.OpenFileDialog(f.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Choose File",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp"},
		},
	})
}

// Upload uploads a file to a server channel.
func (f *FileService) Upload(serverID int64, channelID string, filePath string) (FileInfo, error) {
	conn, err := f.manager.Get(serverID)
	if err != nil {
		return FileInfo{}, fmt.Errorf("get connection: %w", err)
	}

	// Get file info.
	stat, err := os.Stat(filePath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("stat file: %w", err)
	}

	fileName := filepath.Base(filePath)
	mimeType := detectMimeType(filePath)

	// Request upload token via WS.
	payload := map[string]interface{}{
		"name":      fileName,
		"size":      stat.Size(),
		"mime_type": mimeType,
	}
	if channelID != "" {
		payload["channel_id"] = channelID
	}

	resp, err := conn.Request(shared.TypeFileUploadRequest, payload)
	if err != nil {
		return FileInfo{}, fmt.Errorf("file.upload.request: %w", err)
	}

	var tokenResp struct {
		Token     string `json:"token"`
		URL       string `json:"url"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.Unmarshal(resp.Payload, &tokenResp); err != nil {
		return FileInfo{}, fmt.Errorf("unmarshal upload token: %w", err)
	}

	// Build the full upload URL from the server address.
	uploadURL := buildHTTPURL(conn.Address, tokenResp.URL)

	// Upload via HTTP multipart POST.
	file, err := os.Open(filePath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		var sent int64
		buf := make([]byte, 32*1024)
		for {
			n, readErr := file.Read(buf)
			if n > 0 {
				if _, writeErr := part.Write(buf[:n]); writeErr != nil {
					pw.CloseWithError(writeErr)
					return
				}
				sent += int64(n)
				f.emitUploadProgress(fileName, sent, stat.Size(), false, "")
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				pw.CloseWithError(readErr)
				return
			}
		}

		writer.Close()
		pw.Close()
	}()

	req, err := http.NewRequest("POST", uploadURL, pr)
	if err != nil {
		return FileInfo{}, fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokenResp.Token)

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		f.emitUploadProgress(fileName, 0, stat.Size(), true, err.Error())
		return FileInfo{}, fmt.Errorf("upload file: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(httpResp.Body)
		f.emitUploadProgress(fileName, 0, stat.Size(), true, string(body))
		return FileInfo{}, fmt.Errorf("upload failed: %s", httpResp.Status)
	}

	var uploadResult struct {
		FileID string `json:"file_id"`
	}
	json.NewDecoder(httpResp.Body).Decode(&uploadResult)

	f.emitUploadProgress(fileName, stat.Size(), stat.Size(), true, "")

	return FileInfo{
		ID:       uploadResult.FileID,
		FileName: fileName,
		MimeType: mimeType,
		Size:     stat.Size(),
	}, nil
}

// Download downloads a file from a server to a local path.
func (f *FileService) Download(serverID int64, fileID string, savePath string) error {
	conn, err := f.manager.Get(serverID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	// Request download token via WS.
	resp, err := conn.Request(shared.TypeFileDownloadRequest, map[string]interface{}{
		"file_id": fileID,
	})
	if err != nil {
		return fmt.Errorf("file.download.request: %w", err)
	}

	var tokenResp struct {
		Token     string `json:"token"`
		URL       string `json:"url"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.Unmarshal(resp.Payload, &tokenResp); err != nil {
		return fmt.Errorf("unmarshal download token: %w", err)
	}

	downloadURL := buildHTTPURL(conn.Address, tokenResp.URL)

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenResp.Token)

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", httpResp.Status)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	totalSize := httpResp.ContentLength
	var received int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := httpResp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write file: %w", writeErr)
			}
			received += int64(n)
			if totalSize > 0 {
				pct := float64(received) / float64(totalSize) * 100
				emitEvent(f.ctx, "file:downloadProgress", map[string]interface{}{
					"fileId":  fileID,
					"percent": pct,
					"done":    false,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read download: %w", readErr)
		}
	}

	emitEvent(f.ctx, "file:downloadProgress", map[string]interface{}{
		"fileId":  fileID,
		"percent": 100.0,
		"done":    true,
	})
	return nil
}

// PickAndDownload opens a native save dialog and downloads a file.
func (f *FileService) PickAndDownload(serverID int64, fileID string) error {
	savePath, err := wailsRuntime.SaveFileDialog(f.ctx, wailsRuntime.SaveDialogOptions{
		Title: "Save File",
	})
	if err != nil {
		return fmt.Errorf("save dialog: %w", err)
	}
	if savePath == "" {
		return nil
	}
	return f.Download(serverID, fileID, savePath)
}

func (f *FileService) emitUploadProgress(fileName string, sent, total int64, done bool, errMsg string) {
	pct := float64(0)
	if total > 0 {
		pct = float64(sent) / float64(total) * 100
	}
	emitEvent(f.ctx, "file:progress", UploadProgress{
		FileName:  fileName,
		BytesSent: sent,
		TotalSize: total,
		Percent:   pct,
		Done:      done,
		Error:     errMsg,
	})
}

// buildHTTPURL converts a WS server address and relative URL path to a full HTTP URL.
func buildHTTPURL(address, path string) string {
	// Determine scheme based on what the WS connection uses.
	scheme := "https://"
	if isIPAddress(address) {
		scheme = "http://"
	}

	return scheme + address + path
}

// detectMimeType returns a simple MIME type based on file extension.
func detectMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
