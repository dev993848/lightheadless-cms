package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

var mimeToExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// Uploader handles file uploads.
type Uploader struct {
	Dir     string
	MaxSize int64 // bytes; default 5 MB
}

// NewUploader creates a new Uploader for the given directory with a 5 MB max size.
func NewUploader(dir string) *Uploader {
	return &Uploader{
		Dir:     dir,
		MaxSize: 5 * 1024 * 1024, // 5 MB
	}
}

// Save validates and saves an uploaded file.
// Returns the relative path (e.g., "/uploads/abc123.jpg").
func (u *Uploader) Save(fh *multipart.FileHeader) (string, error) {
	if fh.Size > u.MaxSize {
		return "", fmt.Errorf("file size %d exceeds maximum %d bytes", fh.Size, u.MaxSize)
	}

	f, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open upload: %w", err)
	}
	defer f.Close()

	// Read first 512 bytes for MIME detection.
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read upload header: %w", err)
	}
	mime := http.DetectContentType(buf[:n])

	// Normalize: http.DetectContentType may return "image/jpeg; charset=..."
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}

	if !allowedMIME[mime] {
		return "", fmt.Errorf("unsupported MIME type %q; allowed: jpeg, png, gif, webp", mime)
	}

	// Determine extension: prefer from MIME, fall back to original.
	ext := mimeToExt[mime]
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(fh.Filename))
	}

	// Generate unique filename.
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate filename: %w", err)
	}
	filename := hex.EncodeToString(randomBytes) + ext

	// Ensure upload dir exists.
	if err := os.MkdirAll(u.Dir, 0755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

	// Seek back to start before writing.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek upload: %w", err)
	}

	dst, err := os.Create(filepath.Join(u.Dir, filename))
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, f); err != nil {
		return "", fmt.Errorf("write upload: %w", err)
	}

	return "/uploads/" + filename, nil
}

// Delete removes a file by its relative path (e.g., "/uploads/abc123.jpg").
func (u *Uploader) Delete(path string) error {
	if path == "" {
		return nil
	}
	// Strip the leading "/uploads/" prefix and resolve to absolute path.
	name := strings.TrimPrefix(path, "/uploads/")
	if name == "" {
		return nil
	}
	absPath := filepath.Join(u.Dir, name)
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete upload %q: %w", absPath, err)
	}
	return nil
}
