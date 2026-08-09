package notification

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// EmailConfig configures the SMTP sender (Mailpit for local development).
type EmailConfig struct {
	// SMTPHost is the SMTP server host, e.g. "localhost" for Mailpit.
	SMTPHost string
	// SMTPPort is the SMTP server port, e.g. "1025" for Mailpit.
	SMTPPort string
	// From is the envelope sender address.
	From string
}

// EmailSender delivers emails over SMTP. Empty Host means the sender is
// disabled and returns nil (no-op) — used for local runs without Mailpit.
type EmailSender struct {
	host   string
	port   string
	from   string
	enabled bool
}

// NewEmailSender creates a sender; an empty SMTPHost yields a no-op.
func NewEmailSender(cfg EmailConfig) *EmailSender {
	enabled := cfg.SMTPHost != "" && cfg.From != ""
	if !enabled {
		log.Warn().Msg("email sender disabled (SMTP_HOST or SMTP_FROM not set)")
	}
	return &EmailSender{
		host:    cfg.SMTPHost,
		port:    cfg.SMTPPort,
		from:    cfg.From,
		enabled: enabled,
	}
}

// SendEmail sends a plain-text email to the given address.
func (e *EmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	if !e.enabled {
		return nil
	}

	addr := fmt.Sprintf("%s:%s", e.host, e.port)
	// Build with \n separators, then convert once to SMTP's \r\n.
	msg := fmt.Sprintf(
		"From: %s\nTo: %s\nSubject: %s\nDate: %s\nMIME-Version: 1.0\nContent-Type: text/plain; charset=UTF-8\n\n%s",
		e.from, to, subject, time.Now().UTC().Format(time.RFC1123Z), body,
	)
	msg = strings.ReplaceAll(msg, "\n", "\r\n")

	// Dial with a timeout so a dead SMTP server doesn't hang the worker.
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case <-dialCtx.Done():
		return dialCtx.Err()
	default:
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("email: smtp dial: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.Mail(e.from); err != nil {
		return fmt.Errorf("email: mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("email: rcpt: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: data: %w", err)
	}
	if _, err := wc.Write([]byte(msg)); err != nil {
		_ = wc.Close()
		return fmt.Errorf("email: write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("email: close: %w", err)
	}
	return client.Quit()
}
