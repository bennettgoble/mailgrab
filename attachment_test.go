package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIsImageMIME(t *testing.T) {
	tests := []struct {
		mimeType string
		want     bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/svg+xml", true},
		{"IMAGE/JPEG", true},
		{"Image/PNG", true},
		{"text/plain", false},
		{"application/pdf", false},
		{"application/octet-stream", false},
		{"video/mp4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			got := IsImageMIME(tt.mimeType)
			if got != tt.want {
				t.Errorf("IsImageMIME(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestFilterImageAttachments(t *testing.T) {
	attachments := []Attachment{
		{Filename: "photo.jpg", MIMEType: "image/jpeg", Data: bytes.NewReader(nil)},
		{Filename: "document.pdf", MIMEType: "application/pdf", Data: bytes.NewReader(nil)},
		{Filename: "screenshot.png", MIMEType: "image/png", Data: bytes.NewReader(nil)},
		{Filename: "notes.txt", MIMEType: "text/plain", Data: bytes.NewReader(nil)},
		{Filename: "icon.gif", MIMEType: "image/gif", Data: bytes.NewReader(nil)},
	}

	images := FilterImageAttachments(attachments)

	if len(images) != 3 {
		t.Errorf("expected 3 images, got %d", len(images))
	}

	expected := map[string]bool{"photo.jpg": true, "screenshot.png": true, "icon.gif": true}
	for _, img := range images {
		if !expected[img.Filename] {
			t.Errorf("unexpected image in result: %s", img.Filename)
		}
	}
}

func TestFilterImageAttachments_Empty(t *testing.T) {
	var attachments []Attachment
	images := FilterImageAttachments(attachments)
	if images != nil {
		t.Errorf("expected nil, got %v", images)
	}
}

func TestFilterImageAttachments_NoImages(t *testing.T) {
	attachments := []Attachment{
		{Filename: "document.pdf", MIMEType: "application/pdf", Data: bytes.NewReader(nil)},
		{Filename: "notes.txt", MIMEType: "text/plain", Data: bytes.NewReader(nil)},
	}

	images := FilterImageAttachments(attachments)
	if images != nil {
		t.Errorf("expected nil, got %v", images)
	}
}

// MockFileWriter is a test implementation of FileWriter
type MockFileWriter struct {
	WrittenFiles map[string][]byte
	Err          error
}

func (m *MockFileWriter) WriteFile(path string, data []byte) error {
	if m.Err != nil {
		return m.Err
	}
	if m.WrittenFiles == nil {
		m.WrittenFiles = make(map[string][]byte)
	}
	m.WrittenFiles[path] = data
	return nil
}

func TestSaveAttachment(t *testing.T) {
	mockWriter := &MockFileWriter{}
	outputDir := "/tmp/test"
	att := Attachment{
		Filename: "test.jpg",
		MIMEType: "image/jpeg",
		Data:     bytes.NewReader([]byte("fake image data")),
	}

	path, err := SaveAttachment(mockWriter, outputDir, att)
	if err != nil {
		t.Fatalf("SaveAttachment failed: %v", err)
	}

	expectedPath := filepath.Join(outputDir, "test.jpg")
	if path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, path)
	}

	data, ok := mockWriter.WrittenFiles[expectedPath]
	if !ok {
		t.Fatal("file was not written")
	}

	if string(data) != "fake image data" {
		t.Errorf("expected 'fake image data', got %q", string(data))
	}
}

func TestSaveAttachment_RealFS(t *testing.T) {
	tmpDir := t.TempDir()
	fw := OSFileWriter{}

	att := Attachment{
		Filename: "real_test.jpg",
		MIMEType: "image/jpeg",
		Data:     bytes.NewReader([]byte("real fake image data")),
	}

	path, err := SaveAttachment(fw, tmpDir, att)
	if err != nil {
		t.Fatalf("SaveAttachment failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "real_test.jpg")
	if path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if string(data) != "real fake image data" {
		t.Errorf("expected 'real fake image data', got %q", string(data))
	}
}

func TestSaveAttachment_PathTraversal(t *testing.T) {
	tests := []struct {
		name             string
		maliciousName    string
		expectedFilename string
	}{
		{"parent directory", "../../../etc/passwd", "passwd"},
		{"absolute path", "/etc/passwd", "passwd"},
		{"hidden traversal", "foo/../../../etc/passwd", "passwd"},
		{"dot only", ".", "attachment"},
		{"normal filename", "photo.jpg", "photo.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWriter := &MockFileWriter{}
			outputDir := "/tmp/safe"
			att := Attachment{
				Filename: tt.maliciousName,
				MIMEType: "image/jpeg",
				Data:     bytes.NewReader([]byte("data")),
			}

			path, err := SaveAttachment(mockWriter, outputDir, att)
			if err != nil {
				t.Fatalf("SaveAttachment failed: %v", err)
			}

			expectedPath := filepath.Join(outputDir, tt.expectedFilename)
			if path != expectedPath {
				t.Errorf("expected path %q, got %q", expectedPath, path)
			}
		})
	}
}
