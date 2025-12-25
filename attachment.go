package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IsImageMIME returns true if the given MIME type is an image type.
func IsImageMIME(mimeType string) bool {
	return strings.HasPrefix(strings.ToLower(mimeType), "image/")
}

// FileWriter is an interface for writing files, allowing for testing.
type FileWriter interface {
	WriteFile(path string, data []byte) error
}

// OSFileWriter implements FileWriter using the real filesystem.
type OSFileWriter struct{}

func (w OSFileWriter) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// Attachment represents an email attachment.
type Attachment struct {
	Filename string
	MIMEType string
	Data     io.Reader
}

// SaveAttachment saves an attachment to the specified directory.
// Returns the full path where the file was saved, or an error.
func SaveAttachment(fw FileWriter, outputDir string, att Attachment) (string, error) {
	data, err := io.ReadAll(att.Data)
	if err != nil {
		return "", err
	}

	// Sanitize filename to prevent path traversal attacks
	filename := filepath.Base(att.Filename)
	if filename == "." || filename == "/" {
		filename = "attachment"
	}

	path := filepath.Join(outputDir, filename)
	if err := fw.WriteFile(path, data); err != nil {
		return "", err
	}

	return path, nil
}

// FilterImageAttachments returns only attachments with image MIME types.
func FilterImageAttachments(attachments []Attachment) []Attachment {
	var images []Attachment
	for _, att := range attachments {
		if IsImageMIME(att.MIMEType) {
			images = append(images, att)
		}
	}
	return images
}
