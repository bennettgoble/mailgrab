package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

type PostAction string

const (
	PostActionNone   PostAction = "none"
	PostActionDelete PostAction = "delete"
	PostActionMove   PostAction = "move"
)

type Config struct {
	Config     string     `short:"c" long:"config" description:"Path to config file" env:"MAILGRAB_CONFIG"`
	Server     string     `short:"s" long:"server" description:"IMAP server hostname" env:"MAILGRAB_SERVER" yaml:"server" required:"true"`
	Port       int        `short:"p" long:"port" description:"IMAP port" env:"MAILGRAB_PORT" yaml:"port" default:"993"`
	Username   string     `short:"u" long:"username" description:"IMAP username" env:"MAILGRAB_USERNAME" yaml:"username" required:"true"`
	Password   string     `short:"P" long:"password" description:"IMAP password" env:"MAILGRAB_PASSWORD" yaml:"password" required:"true"`
	Mailbox    string     `short:"m" long:"mailbox" description:"Mailbox to check" env:"MAILGRAB_MAILBOX" yaml:"mailbox" default:"Inbox"`
	Output     string     `short:"o" long:"output" description:"Output directory for attachments" env:"MAILGRAB_OUTPUT" yaml:"output" required:"true"`
	PostAction PostAction `long:"post-action" description:"Action after processing: none, delete, move" env:"MAILGRAB_POST_ACTION" yaml:"post_action" default:"none"`
	MoveTo     string     `long:"move-to" description:"Target folder for move action" env:"MAILGRAB_MOVE_TO" yaml:"move_to"`
	Insecure   bool       `long:"insecure" description:"Disable TLS verification" env:"MAILGRAB_INSECURE" yaml:"insecure"`
	Verbose    bool       `short:"v" long:"verbose" description:"Enable verbose output" env:"MAILGRAB_VERBOSE" yaml:"verbose"`
	Quiet      bool       `short:"q" long:"quiet" description:"Suppress non-error output" env:"MAILGRAB_QUIET" yaml:"quiet"`
	JSONOutput string     `short:"j" long:"json-output" description:"Path to JSON output file" env:"MAILGRAB_JSON_OUTPUT" yaml:"json_output"`
}

func (c *Config) Validate() error {
	if c.PostAction == PostActionMove && c.MoveTo == "" {
		return errors.New("move_to is required when post_action is 'move'")
	}
	if c.Verbose && c.Quiet {
		return errors.New("verbose and quiet cannot both be set")
	}
	switch c.PostAction {
	case PostActionNone, PostActionDelete, PostActionMove, "":
	default:
		return fmt.Errorf("invalid post_action: %s (must be none, delete, or move)", c.PostAction)
	}
	return nil
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// First pass: parse only to get config file path (ignore errors for missing required fields)
	parser := flags.NewParser(cfg, flags.IgnoreUnknown)
	if _, err := parser.Parse(); err != nil {
		// Ignore errors in first pass - we just want the config file path
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
	}

	// Load from config file if specified or found
	configFile, err := findConfigFile(cfg.Config)
	if err != nil {
		return nil, err
	}
	if configFile != "" {
		if err := loadConfigFile(configFile, cfg); err != nil {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
	}

	// Second pass: parse again with full validation
	// Environment variables and flags will override config file values
	parser = flags.NewParser(cfg, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		return nil, err
	}

	// Set default for empty post_action
	if cfg.PostAction == "" {
		cfg.PostAction = PostActionNone
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func findConfigFile(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config file not found: %s", explicit)
		}
		return explicit, nil
	}

	// Check ./mailgrab.yaml
	if _, err := os.Stat("mailgrab.yaml"); err == nil {
		return "mailgrab.yaml", nil
	}

	// Check ~/.config/mailgrab/config.yaml
	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".config", "mailgrab", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	return "", nil
}

func loadConfigFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}
