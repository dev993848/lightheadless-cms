package services

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewUploader(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir)
	if u.Dir != dir {
		t.Errorf("expected dir %q, got %q", dir, u.Dir)
	}
	if u.MaxSize != 5*1024*1024 {
		t.Errorf("expected MaxSize=5MB, got %d", u.MaxSize)
	}
}

// makeFakeImage builds a minimal 1x1 PNG in memory (actual PNG header bytes).
func makeFakeImage() []byte {
	// Minimal valid PNG bytes (1x1 pixel).
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func makeFakeMultipartFile(t *testing.T, content []byte, filename, contentType string) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="image"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	part.Write(content)
	w.Close()

	reader := multipart.NewReader(&buf, w.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		t.Fatalf("read form: %v", err)
	}
	files := form.File["image"]
	if len(files) == 0 {
		t.Fatal("no files in form")
	}
	return files[0]
}

func TestUploader_Save_PNG(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir)

	fh := makeFakeMultipartFile(t, makeFakeImage(), "test.png", "image/png")
	path, err := u.Save(fh)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasPrefix(path, "/uploads/") {
		t.Errorf("expected path starting with /uploads/, got %q", path)
	}
	if !strings.HasSuffix(path, ".png") {
		t.Errorf("expected .png extension, got %q", path)
	}

	// Verify file exists on disk.
	filename := strings.TrimPrefix(path, "/uploads/")
	if _, err := os.Stat(filepath.Join(dir, filename)); os.IsNotExist(err) {
		t.Error("file should exist on disk after Save")
	}
}

func TestUploader_Save_TooBig(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir)
	u.MaxSize = 10 // 10 bytes max

	bigContent := make([]byte, 100)
	fh := makeFakeMultipartFile(t, bigContent, "big.jpg", "image/jpeg")
	fh.Size = 100 // override size check

	_, err := u.Save(fh)
	if err == nil {
		t.Error("expected error for oversized file")
	}
}

func TestUploader_Save_InvalidMIME(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir)

	// Plain text content — not an image.
	fh := makeFakeMultipartFile(t, []byte("hello world this is text"), "test.txt", "text/plain")
	_, err := u.Save(fh)
	if err == nil {
		t.Error("expected error for non-image MIME type")
	}
}

func TestUploader_Delete_Existing(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir)

	// Create a real file.
	fname := "testfile.png"
	if err := os.WriteFile(filepath.Join(dir, fname), []byte("data"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := u.Delete("/uploads/" + fname); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, fname)); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestUploader_Delete_NotExists(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir)

	// Deleting a non-existent file should not return error.
	if err := u.Delete("/uploads/nonexistent.png"); err != nil {
		t.Errorf("Delete of non-existent file should not error: %v", err)
	}
}

func TestUploader_Delete_EmptyPath(t *testing.T) {
	dir := t.TempDir()
	u := NewUploader(dir)

	if err := u.Delete(""); err != nil {
		t.Errorf("Delete of empty path should not error: %v", err)
	}
}
