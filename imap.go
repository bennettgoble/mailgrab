package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

const seenKeyword imap.Flag = "mailgrab-seen"

// IMAPClient defines the interface for IMAP operations, enabling testing.
type IMAPClient interface {
	Login(username, password string) *imapclient.Command
	Select(mailbox string, options *imap.SelectOptions) *imapclient.SelectCommand
	Search(criteria *imap.SearchCriteria, options *imap.SearchOptions) *imapclient.SearchCommand
	Fetch(numSet imap.NumSet, options *imap.FetchOptions) *imapclient.FetchCommand
	Store(numSet imap.NumSet, store *imap.StoreFlags, options *imap.StoreOptions) *imapclient.FetchCommand
	Move(numSet imap.NumSet, mailbox string) *imapclient.MoveCommand
	UIDExpunge(uids imap.UIDSet) *imapclient.ExpungeCommand
	Logout() *imapclient.Command
	Close() error
}

// Message represents a fetched email message with its attachments.
type Message struct {
	UID         imap.UID
	Subject     string
	Attachments []Attachment
}

// MailClient wraps IMAP operations for mailgrab.
type MailClient struct {
	client  IMAPClient
	cfg     *Config
	verbose func(string, ...any)
}

// NewMailClient creates a new MailClient connected to the IMAP server.
func NewMailClient(cfg *Config, verbose func(string, ...any)) (*MailClient, error) {
	var client *imapclient.Client
	var err error

	addr := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)

	options := &imapclient.Options{}
	if cfg.Insecure {
		options.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	verbose("Connecting to %s...", addr)
	client, err = imapclient.DialTLS(addr, options)
	if err != nil {
		return nil, fmt.Errorf("connecting to server: %w", err)
	}

	verbose("Authenticating as %s...", cfg.Username)
	if err := client.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return &MailClient{
		client:  client,
		cfg:     cfg,
		verbose: verbose,
	}, nil
}

// FetchNewMessages fetches all messages that haven't been processed yet.
func (m *MailClient) FetchNewMessages() ([]Message, error) {
	m.verbose("Selecting mailbox: %s", m.cfg.Mailbox)
	if _, err := m.client.Select(m.cfg.Mailbox, nil).Wait(); err != nil {
		return nil, fmt.Errorf("selecting mailbox: %w", err)
	}

	// Search for messages without our custom keyword
	criteria := &imap.SearchCriteria{
		NotFlag: []imap.Flag{seenKeyword},
	}
	searchData, err := m.client.Search(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("searching messages: %w", err)
	}

	seqNums := searchData.AllSeqNums()
	if len(seqNums) == 0 {
		m.verbose("No new messages found")
		return nil, nil
	}
	m.verbose("Found %d new message(s)", len(seqNums))

	seqSet := imap.SeqSetNum(seqNums...)
	fetchOptions := &imap.FetchOptions{
		UID:           true,
		Envelope:      true,
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	fetchCmd := m.client.Fetch(seqSet, fetchOptions)
	defer func() { _ = fetchCmd.Close() }()

	var messages []Message
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		var uid imap.UID
		var subject string
		var bodyStructure imap.BodyStructure

		for {
			item := msg.Next()
			if item == nil {
				break
			}
			switch data := item.(type) {
			case imapclient.FetchItemDataUID:
				uid = data.UID
			case imapclient.FetchItemDataEnvelope:
				subject = data.Envelope.Subject
			case imapclient.FetchItemDataBodyStructure:
				bodyStructure = data.BodyStructure
			}
		}

		var attachments []Attachment
		if bodyStructure != nil {
			var err error
			attachments, err = m.fetchAttachments(uid, bodyStructure)
			if err != nil {
				return nil, fmt.Errorf("fetching attachments for UID %d: %w", uid, err)
			}
		}

		messages = append(messages, Message{
			UID:         uid,
			Subject:     subject,
			Attachments: attachments,
		})
	}

	return messages, nil
}

// fetchAttachments fetches attachment data from a message.
func (m *MailClient) fetchAttachments(uid imap.UID, bs imap.BodyStructure) ([]Attachment, error) {
	parts := findAttachmentParts(bs, nil)
	if len(parts) == 0 {
		return nil, nil
	}

	var attachments []Attachment
	for _, part := range parts {
		att, err := m.fetchPart(uid, part)
		if err != nil {
			return nil, err
		}
		if att != nil {
			attachments = append(attachments, *att)
		}
	}

	return attachments, nil
}

type attachmentPart struct {
	path     []int
	mimeType string
	filename string
}

// findAttachmentParts recursively finds all attachment parts in a body structure.
func findAttachmentParts(bs imap.BodyStructure, path []int) []attachmentPart {
	var parts []attachmentPart

	switch s := bs.(type) {
	case *imap.BodyStructureSinglePart:
		// Check if it's an attachment with a filename
		filename := ""
		if disp := s.Disposition(); disp != nil && disp.Params != nil {
			filename = disp.Params["filename"]
		}
		if filename == "" && s.Params != nil {
			filename = s.Params["name"]
		}

		if filename != "" {
			mimeType := strings.ToLower(s.Type) + "/" + strings.ToLower(s.Subtype)
			parts = append(parts, attachmentPart{
				path:     append([]int{}, path...),
				mimeType: mimeType,
				filename: filename,
			})
		}

	case *imap.BodyStructureMultiPart:
		for i, child := range s.Children {
			childPath := append(path, i+1)
			parts = append(parts, findAttachmentParts(child, childPath)...)
		}
	}

	return parts
}

// fetchPart fetches a specific MIME part from a message.
func (m *MailClient) fetchPart(uid imap.UID, part attachmentPart) (*Attachment, error) {
	// Build section path
	var section imap.FetchItemBodySection
	section.Part = part.path

	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{&section},
	}

	uidSet := imap.UIDSetNum(uid)
	fetchCmd := m.client.Fetch(uidSet, fetchOptions)
	defer func() { _ = fetchCmd.Close() }()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("message not found")
	}

	var data []byte
	for {
		item := msg.Next()
		if item == nil {
			break
		}
		if bodySection, ok := item.(imapclient.FetchItemDataBodySection); ok {
			var err error
			data, err = io.ReadAll(bodySection.Literal)
			if err != nil {
				return nil, fmt.Errorf("reading body section: %w", err)
			}
		}
	}

	if data == nil {
		return nil, nil
	}

	return &Attachment{
		Filename: part.filename,
		MIMEType: part.mimeType,
		Data:     bytes.NewReader(data),
	}, nil
}

// MarkProcessed marks a message as processed by adding our custom keyword.
func (m *MailClient) MarkProcessed(uid imap.UID) error {
	uidSet := imap.UIDSetNum(uid)

	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{seenKeyword},
		Silent: true,
	}

	if err := m.client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("marking message as processed: %w", err)
	}

	return nil
}

// DeleteMessage deletes a message by UID.
func (m *MailClient) DeleteMessage(uid imap.UID) error {
	uidSet := imap.UIDSetNum(uid)

	storeFlags := &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Flags:  []imap.Flag{imap.FlagDeleted},
		Silent: true,
	}

	if err := m.client.Store(uidSet, storeFlags, nil).Close(); err != nil {
		return fmt.Errorf("marking message as deleted: %w", err)
	}

	if err := m.client.UIDExpunge(uidSet).Close(); err != nil {
		return fmt.Errorf("expunging message: %w", err)
	}

	return nil
}

// MoveMessage moves a message to another mailbox.
func (m *MailClient) MoveMessage(uid imap.UID, destMailbox string) error {
	uidSet := imap.UIDSetNum(uid)

	if _, err := m.client.Move(uidSet, destMailbox).Wait(); err != nil {
		return fmt.Errorf("moving message: %w", err)
	}

	return nil
}

// Close closes the IMAP connection.
func (m *MailClient) Close() error {
	_ = m.client.Logout().Wait()
	return m.client.Close()
}
