package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "move action without move_to",
			cfg:     Config{Server: "imap.example.com", Username: "user", Password: "pass", Output: "/tmp", PostAction: PostActionMove},
			wantErr: "move_to is required when post_action is 'move'",
		},
		{
			name:    "verbose and quiet both set",
			cfg:     Config{Server: "imap.example.com", Username: "user", Password: "pass", Output: "/tmp", Verbose: true, Quiet: true},
			wantErr: "verbose and quiet cannot both be set",
		},
		{
			name:    "invalid post_action",
			cfg:     Config{Server: "imap.example.com", Username: "user", Password: "pass", Output: "/tmp", PostAction: "invalid"},
			wantErr: "invalid post_action: invalid",
		},
		{
			name: "valid config with defaults",
			cfg:  Config{Server: "imap.example.com", Username: "user", Password: "pass", Output: "/tmp"},
		},
		{
			name: "valid config with move action",
			cfg:  Config{Server: "imap.example.com", Username: "user", Password: "pass", Output: "/tmp", PostAction: PostActionMove, MoveTo: "Archive"},
		},
		{
			name: "valid config with delete action",
			cfg:  Config{Server: "imap.example.com", Username: "user", Password: "pass", Output: "/tmp", PostAction: PostActionDelete},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got %q", err.Error())
			}
		})
	}
}

func TestLoadConfigFile(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `server: imap.example.com
port: 993
username: testuser
password: testpass
mailbox: INBOX
output: /tmp/attachments
post_action: none
verbose: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg := &Config{}
	if err := loadConfigFile(configPath, cfg); err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	if cfg.Server != "imap.example.com" {
		t.Errorf("expected server 'imap.example.com', got %q", cfg.Server)
	}
	if cfg.Port != 993 {
		t.Errorf("expected port 993, got %d", cfg.Port)
	}
	if cfg.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("expected password 'testpass', got %q", cfg.Password)
	}
	if cfg.Mailbox != "INBOX" {
		t.Errorf("expected mailbox 'INBOX', got %q", cfg.Mailbox)
	}
	if cfg.Output != "/tmp/attachments" {
		t.Errorf("expected output '/tmp/attachments', got %q", cfg.Output)
	}
	if cfg.PostAction != PostActionNone {
		t.Errorf("expected post_action 'none', got %q", cfg.PostAction)
	}
	if !cfg.Verbose {
		t.Errorf("expected verbose true, got false")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Test with explicit path that exists
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom.yaml")
	if err := os.WriteFile(configPath, []byte("server: test"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	found, err := findConfigFile(configPath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if found != configPath {
		t.Errorf("expected %q, got %q", configPath, found)
	}

	// Test with non-existent explicit path - should error
	found, err = findConfigFile("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for non-existent explicit path, got nil")
	}
	if found != "" {
		t.Errorf("expected empty string for non-existent path, got %q", found)
	}

	// Test with no explicit path - should return empty without error
	_, err = findConfigFile("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Result depends on whether mailgrab.yaml exists in current dir
}
