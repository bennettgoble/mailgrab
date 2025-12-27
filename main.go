package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	exitOK           = 0
	exitConfigError  = 1
	exitConnectError = 2
	exitProcessError = 3
)

// JSONMessageOutput represents a message in JSON output
type JSONMessageOutput struct {
	From    string   `json:"from"`
	Subject string   `json:"subject"`
	Images  []string `json:"images"`
}

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return exitConfigError
	}

	// Set up logging based on verbosity
	verbose := func(format string, args ...any) {}
	if cfg.Verbose {
		verbose = func(format string, args ...any) {
			fmt.Printf(format+"\n", args...)
		}
	}

	// Connect to IMAP server
	client, err := NewMailClient(cfg, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return exitConnectError
	}
	defer func() { _ = client.Close() }()

	// Fetch new messages
	messages, err := client.FetchNewMessages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return exitProcessError
	}

	if len(messages) == 0 {
		if !cfg.Quiet {
			fmt.Println("No new messages")
		}
		return exitOK
	}

	verbose("Processing %d message(s)...", len(messages))

	fileWriter := OSFileWriter{}
	totalSaved := 0
	outputDirCreated := false
	var jsonOutput []JSONMessageOutput

	for _, msg := range messages {
		// Filter to only image attachments
		images := FilterImageAttachments(msg.Attachments)

		savedCount := 0
		var savedFilenames []string
		for _, att := range images {
			// Create output directory on first image save
			if !outputDirCreated {
				if err := os.MkdirAll(cfg.Output, 0755); err != nil {
					fmt.Fprintf(os.Stderr, "Error: creating output directory: %v\n", err)
					return exitConfigError
				}
				outputDirCreated = true
			}

			path, err := SaveAttachment(fileWriter, cfg.Output, att)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: saving attachment %s: %v\n", att.Filename, err)
				continue
			}
			verbose("  Saved: %s", path)
			savedCount++
			savedFilenames = append(savedFilenames, att.Filename)
		}

		// Add to JSON output if images were saved
		if len(savedFilenames) > 0 {
			jsonOutput = append(jsonOutput, JSONMessageOutput{
				From:    msg.From,
				Subject: msg.Subject,
				Images:  savedFilenames,
			})
		}

		if cfg.Verbose {
			if savedCount > 0 {
				fmt.Printf("  Message %d: %q - saved %d image(s)\n", msg.UID, msg.Subject, savedCount)
			} else {
				fmt.Printf("  Message %d: %q - no image attachments\n", msg.UID, msg.Subject)
			}
		}

		totalSaved += savedCount

		// Mark message as processed
		if err := client.MarkProcessed(msg.UID); err != nil {
			fmt.Fprintf(os.Stderr, "Error: marking message as processed: %v\n", err)
		}

		// Perform post-action
		switch cfg.PostAction {
		case PostActionDelete:
			if err := client.DeleteMessage(msg.UID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: deleting message: %v\n", err)
			}
		case PostActionMove:
			if err := client.MoveMessage(msg.UID, cfg.MoveTo); err != nil {
				fmt.Fprintf(os.Stderr, "Error: moving message: %v\n", err)
			}
		}
	}

	if !cfg.Quiet {
		fmt.Printf("Processed %d message(s), saved %d image(s)\n", len(messages), totalSaved)
	}

	// Write JSON output if configured
	if cfg.JSONOutput != "" && len(jsonOutput) > 0 {
		if err := writeJSONOutput(cfg.JSONOutput, jsonOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error: writing JSON output: %v\n", err)
			// Don't fail - JSON is supplementary
		}
	}

	return exitOK
}

// writeJSONOutput writes processing results to a JSON file
func writeJSONOutput(path string, data []JSONMessageOutput) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
