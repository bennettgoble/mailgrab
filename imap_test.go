package main

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func TestFindAttachmentParts_SinglePart(t *testing.T) {
	bs := &imap.BodyStructureSinglePart{
		Type:    "IMAGE",
		Subtype: "JPEG",
		Params:  map[string]string{"name": "photo.jpg"},
	}

	parts := findAttachmentParts(bs, nil)

	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}

	if parts[0].filename != "photo.jpg" {
		t.Errorf("expected filename 'photo.jpg', got %q", parts[0].filename)
	}
	if parts[0].mimeType != "image/jpeg" {
		t.Errorf("expected mimeType 'image/jpeg', got %q", parts[0].mimeType)
	}
}

func TestFindAttachmentParts_MultiPart(t *testing.T) {
	bs := &imap.BodyStructureMultiPart{
		Children: []imap.BodyStructure{
			&imap.BodyStructureSinglePart{
				Type:    "TEXT",
				Subtype: "PLAIN",
			},
			&imap.BodyStructureSinglePart{
				Type:    "IMAGE",
				Subtype: "PNG",
				Params:  map[string]string{"name": "screenshot.png"},
			},
			&imap.BodyStructureSinglePart{
				Type:    "APPLICATION",
				Subtype: "PDF",
				Params:  map[string]string{"name": "document.pdf"},
			},
		},
	}

	parts := findAttachmentParts(bs, nil)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts (attachments with filenames), got %d", len(parts))
	}

	// Check screenshot.png
	found := false
	for _, p := range parts {
		if p.filename == "screenshot.png" {
			found = true
			if p.mimeType != "image/png" {
				t.Errorf("expected mimeType 'image/png', got %q", p.mimeType)
			}
			if len(p.path) != 1 || p.path[0] != 2 {
				t.Errorf("expected path [2], got %v", p.path)
			}
		}
	}
	if !found {
		t.Error("screenshot.png not found in parts")
	}

	// Check document.pdf
	found = false
	for _, p := range parts {
		if p.filename == "document.pdf" {
			found = true
			if p.mimeType != "application/pdf" {
				t.Errorf("expected mimeType 'application/pdf', got %q", p.mimeType)
			}
		}
	}
	if !found {
		t.Error("document.pdf not found in parts")
	}
}

func TestFindAttachmentParts_Nested(t *testing.T) {
	bs := &imap.BodyStructureMultiPart{
		Children: []imap.BodyStructure{
			&imap.BodyStructureSinglePart{
				Type:    "TEXT",
				Subtype: "PLAIN",
			},
			&imap.BodyStructureMultiPart{
				Children: []imap.BodyStructure{
					&imap.BodyStructureSinglePart{
						Type:    "TEXT",
						Subtype: "HTML",
					},
					&imap.BodyStructureSinglePart{
						Type:    "IMAGE",
						Subtype: "GIF",
						Params:  map[string]string{"name": "inline.gif"},
					},
				},
			},
			&imap.BodyStructureSinglePart{
				Type:    "IMAGE",
				Subtype: "JPEG",
				Params:  map[string]string{"name": "attached.jpg"},
			},
		},
	}

	parts := findAttachmentParts(bs, nil)

	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}

	// Check inline.gif - should be at path [2, 2]
	found := false
	for _, p := range parts {
		if p.filename == "inline.gif" {
			found = true
			if len(p.path) != 2 || p.path[0] != 2 || p.path[1] != 2 {
				t.Errorf("expected path [2, 2], got %v", p.path)
			}
		}
	}
	if !found {
		t.Error("inline.gif not found")
	}

	// Check attached.jpg - should be at path [3]
	found = false
	for _, p := range parts {
		if p.filename == "attached.jpg" {
			found = true
			if len(p.path) != 1 || p.path[0] != 3 {
				t.Errorf("expected path [3], got %v", p.path)
			}
		}
	}
	if !found {
		t.Error("attached.jpg not found")
	}
}

func TestFindAttachmentParts_NoAttachments(t *testing.T) {
	bs := &imap.BodyStructureSinglePart{
		Type:    "TEXT",
		Subtype: "PLAIN",
	}

	parts := findAttachmentParts(bs, nil)

	if len(parts) != 0 {
		t.Errorf("expected 0 parts, got %d", len(parts))
	}
}
