package services

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"os"

	"haven/client/connection"
	havenCrypto "haven/client/crypto"
	"haven/client/keystore"
	"haven/client/models"

	"gorm.io/gorm"

	_ "golang.org/x/image/webp"
)

// Profile is the frontend-facing profile data.
type Profile struct {
	PublicKey   string `json:"publicKey"`
	DisplayName string `json:"displayName"`
	AvatarHash  string `json:"avatarHash"`
	Bio         string `json:"bio"`
}

// ProfileService manages the local user profile.
type ProfileService struct {
	ctx     context.Context
	db      *gorm.DB
	manager *connection.Manager
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey
}

// NewProfileService creates a new ProfileService.
func NewProfileService(db *gorm.DB, manager *connection.Manager, privKey ed25519.PrivateKey) *ProfileService {
	return &ProfileService{
		db:      db,
		manager: manager,
		privKey: privKey,
		pubKey:  privKey.Public().(ed25519.PublicKey),
	}
}

// SetContext is called by Wails during startup.
func (p *ProfileService) SetContext(ctx context.Context) {
	p.ctx = ctx
}

// GetProfile returns the local profile.
func (p *ProfileService) GetProfile() (Profile, error) {
	var lp models.LocalProfile
	if err := p.db.First(&lp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return Profile{
				PublicKey: havenCrypto.HexEncode(p.pubKey),
			}, nil
		}
		return Profile{}, fmt.Errorf("get profile: %w", err)
	}

	bio := ""
	if lp.Bio != nil {
		bio = *lp.Bio
	}
	avatarHash := ""
	if lp.AvatarHash != nil {
		avatarHash = *lp.AvatarHash
	}

	return Profile{
		PublicKey:   havenCrypto.HexEncode(lp.PublicKey),
		DisplayName: lp.DisplayName,
		AvatarHash:  avatarHash,
		Bio:         bio,
	}, nil
}

// UpdateProfile updates the local profile display name, bio, and optionally avatar.
// Pass an empty avatarPath to skip avatar update.
func (p *ProfileService) UpdateProfile(displayName string, bio string, avatarPath string) error {
	var lp models.LocalProfile
	err := p.db.First(&lp).Error
	isNew := err == gorm.ErrRecordNotFound

	if isNew {
		lp = models.LocalProfile{
			PublicKey:   p.pubKey,
			DisplayName: displayName,
			Bio:         &bio,
		}
		if err := p.db.Create(&lp).Error; err != nil {
			return fmt.Errorf("create profile: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("get profile: %w", err)
	} else {
		lp.DisplayName = displayName
		lp.Bio = &bio
		if err := p.db.Save(&lp).Error; err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
	}

	// Process avatar if a path was provided.
	if avatarPath != "" {
		imgData, err := processAvatarFile(avatarPath)
		if err != nil {
			return fmt.Errorf("process avatar: %w", err)
		}
		hash := sha256.Sum256(imgData)
		hashStr := hex.EncodeToString(hash[:])
		lp.Avatar = imgData
		lp.AvatarHash = &hashStr
		if err := p.db.Save(&lp).Error; err != nil {
			return fmt.Errorf("save avatar: %w", err)
		}
	}

	if isNew {
		// First-time setup complete — transition app to ready.
		emitEvent(p.ctx, "app:stateChanged", AppState{
			Phase:    "ready",
			Progress: 100,
		})
	}
	return nil
}

const maxAvatarSize = 256 // px — avatars are resized to fit within this

// ReadFileAsDataURL reads a file and returns it as a base64 data URL with no processing.
// Used for instant preview before the heavier ProcessAvatar runs.
func (p *ProfileService) ReadFileAsDataURL(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		// ok
	default:
		return "", fmt.Errorf("unsupported image format: %s", mime)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mime, b64), nil
}

// PreviewAvatar reads an image file, validates it, resizes it, and returns
// a base64 data URL. This is the processed/final version.
func (p *ProfileService) PreviewAvatar(filePath string) (string, error) {
	imgData, err := processAvatarFile(filePath)
	if err != nil {
		return "", err
	}
	mime := http.DetectContentType(imgData)
	b64 := base64.StdEncoding.EncodeToString(imgData)
	return fmt.Sprintf("data:%s;base64,%s", mime, b64), nil
}

// SetAvatar reads an image file, resizes it, and stores it on the profile.
// The profile row must already exist.
func (p *ProfileService) SetAvatar(filePath string) error {
	imgData, err := processAvatarFile(filePath)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(imgData)
	hashStr := hex.EncodeToString(hash[:])

	var lp models.LocalProfile
	if err := p.db.First(&lp).Error; err != nil {
		return fmt.Errorf("get profile: %w", err)
	}

	lp.Avatar = imgData
	lp.AvatarHash = &hashStr
	if err := p.db.Save(&lp).Error; err != nil {
		return fmt.Errorf("update avatar: %w", err)
	}
	return nil
}

// GetAvatar returns the stored avatar as a base64 data URL, or empty string if none.
func (p *ProfileService) GetAvatar() (string, error) {
	var lp models.LocalProfile
	if err := p.db.First(&lp).Error; err != nil {
		return "", nil
	}
	if len(lp.Avatar) == 0 {
		return "", nil
	}
	mime := http.DetectContentType(lp.Avatar)
	b64 := base64.StdEncoding.EncodeToString(lp.Avatar)
	return fmt.Sprintf("data:%s;base64,%s", mime, b64), nil
}

// processAvatarFile reads an image, validates the format, resizes to maxAvatarSize,
// and returns the processed bytes. GIFs are kept as animated GIFs; everything else
// becomes PNG.
func processAvatarFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read avatar file: %w", err)
	}

	// Validate it's actually an image we support.
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		// ok
	default:
		return nil, fmt.Errorf("unsupported image format: %s", mime)
	}

	// GIFs get special handling to preserve animation.
	if mime == "image/gif" {
		return processGIF(data)
	}

	return processStaticImage(data)
}

// processStaticImage decodes a PNG/JPEG/WebP, resizes, and returns PNG bytes.
func processStaticImage(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() > maxAvatarSize || bounds.Dy() > maxAvatarSize {
		img = resizeImage(img, maxAvatarSize)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode avatar png: %w", err)
	}
	return buf.Bytes(), nil
}

// processGIF decodes an animated GIF, resizes all frames, and returns GIF bytes.
func processGIF(data []byte) ([]byte, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode gif: %w", err)
	}

	// Determine scale factor from the overall GIF dimensions.
	srcW, srcH := g.Config.Width, g.Config.Height
	if srcW <= maxAvatarSize && srcH <= maxAvatarSize {
		// Already small enough, but cap total size to avoid storing huge GIFs.
		if len(data) > 2*1024*1024 {
			return nil, fmt.Errorf("GIF too large (max 2 MB)")
		}
		return data, nil
	}

	dstW, dstH := fitDimensions(srcW, srcH, maxAvatarSize)
	g.Config.Width = dstW
	g.Config.Height = dstH

	for i, frame := range g.Image {
		// Compose frame onto a full-size canvas to handle disposal correctly.
		canvas := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		// Resize the full canvas.
		resized := resizeImage(canvas, maxAvatarSize)

		// Convert back to paletted image.
		palettedFrame := image.NewPaletted(resized.Bounds(), frame.Palette)
		draw.FloydSteinberg.Draw(palettedFrame, resized.Bounds(), resized, image.Point{})
		g.Image[i] = palettedFrame
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, fmt.Errorf("encode avatar gif: %w", err)
	}

	// Safety check — if the resized GIF is still huge, reject it.
	if buf.Len() > 2*1024*1024 {
		return nil, fmt.Errorf("GIF too large after resize (max 2 MB)")
	}
	return buf.Bytes(), nil
}

// fitDimensions calculates new dimensions that fit within maxDim while preserving aspect ratio.
func fitDimensions(srcW, srcH, maxDim int) (int, int) {
	if srcW >= srcH {
		return maxDim, max(1, srcH*maxDim/srcW)
	}
	return max(1, srcW*maxDim/srcH), maxDim
}

// resizeImage scales an image so its largest dimension is maxDim, using nearest-neighbor.
func resizeImage(src image.Image, maxDim int) *image.RGBA {
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	dstW, dstH := fitDimensions(srcW, srcH, maxDim)

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		srcY := bounds.Min.Y + y*srcH/dstH
		for x := 0; x < dstW; x++ {
			srcX := bounds.Min.X + x*srcW/dstW
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

// RemoveAvatar removes the local profile avatar.
func (p *ProfileService) RemoveAvatar() error {
	var lp models.LocalProfile
	if err := p.db.First(&lp).Error; err != nil {
		return fmt.Errorf("get profile: %w", err)
	}

	lp.Avatar = nil
	lp.AvatarHash = nil
	if err := p.db.Save(&lp).Error; err != nil {
		return fmt.Errorf("remove avatar: %w", err)
	}
	return nil
}

// GetPublicKey returns the hex-encoded Ed25519 public key.
func (p *ProfileService) GetPublicKey() string {
	return havenCrypto.HexEncode(p.pubKey)
}

// ExportIdentity exports the private key to a file.
func (p *ProfileService) ExportIdentity(filePath string) error {
	encoded := hex.EncodeToString(p.privKey)
	if err := os.WriteFile(filePath, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("export identity: %w", err)
	}
	return nil
}

// ImportIdentity imports a private key from a file and replaces the current identity.
func (p *ProfileService) ImportIdentity(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read identity file: %w", err)
	}

	keyBytes, err := hex.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("decode identity: %w", err)
	}

	if len(keyBytes) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid key length: %d", len(keyBytes))
	}

	newKey := ed25519.PrivateKey(keyBytes)
	if err := keystore.Store(newKey); err != nil {
		return fmt.Errorf("store imported identity: %w", err)
	}

	return nil
}
