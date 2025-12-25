package main

import (
	"fmt"
	"os"
)

const (
	exitOK           = 0
	exitConfigError  = 1
	exitConnectError = 2
	exitProcessError = 3
)

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

	for _, msg := range messages {
		// Filter to only image attachments
		images := FilterImageAttachments(msg.Attachments)

		savedCount := 0
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

	return exitOK
}
