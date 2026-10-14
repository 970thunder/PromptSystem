package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"promptos-backend/internal/config"
	"promptos-backend/internal/store"
)

type emailSender interface {
	Send(ctx context.Context, recipient, subject, body string) error
}

type smtpEmailSender struct {
	host, port, username, password, from string
}

func newSMTPEmailSender(cfg config.Config) emailSender {
	if strings.TrimSpace(cfg.SMTPHost) == "" || strings.TrimSpace(cfg.SMTPFrom) == "" {
		return nil
	}
	return &smtpEmailSender{host: cfg.SMTPHost, port: cfg.SMTPPort, username: cfg.SMTPUser, password: cfg.SMTPPassword, from: cfg.SMTPFrom}
}

// newSMTPSenderFromSettings builds a one-shot sender from typed settings. It
// returns nil when the settings do not carry both required fields.
func newSMTPSenderFromSettings(s store.SiteSettings) *smtpEmailSender {
	if !s.SMTPConfigured() {
		return nil
	}
	port := s.SMTPPort
	if port == "" {
		port = "587"
	}
	return &smtpEmailSender{host: s.SMTPHost, port: port, username: s.SMTPUser, password: s.SMTPPassword, from: s.SMTPFrom}
}

// settingsEmailSender resolves the effective sender per send: wizard/admin
// settings override environment configuration, so SMTP can be configured at
// runtime without a restart. The settings map is cached until Invalidate is
// called by the write paths (admin console, setup wizard) — the hot
// "is email configured" check no longer hits the settings table per request,
// while an actual Send always resolves through the same cache warmed by the
// most recent write.
type settingsEmailSender struct {
	settings store.SettingsManager
	fallback emailSender

	mu    sync.RWMutex
	cache *store.SiteSettings
}

func newSettingsEmailSender(settings store.SettingsManager, cfg config.Config) emailSender {
	return &settingsEmailSender{settings: settings, fallback: newSMTPEmailSender(cfg)}
}

// Invalidate drops the cached settings snapshot; the next resolve re-reads.
func (s *settingsEmailSender) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = nil
}

func (s *settingsEmailSender) current() store.SiteSettings {
	s.mu.RLock()
	cached := s.cache
	s.mu.RUnlock()
	if cached != nil {
		return *cached
	}
	var all map[string]string
	if s.settings != nil {
		if read, err := s.settings.GetAll(); err == nil {
			all = read
		}
	}
	settings := store.ParseSiteSettings(all)
	s.mu.Lock()
	s.cache = &settings
	s.mu.Unlock()
	return settings
}

func (s *settingsEmailSender) resolve() emailSender {
	if s == nil {
		return nil
	}
	if sender := newSMTPSenderFromSettings(s.current()); sender != nil {
		return sender
	}
	return s.fallback
}

// Configured reports whether any sender source is usable, replacing the old
// `emailSender == nil` checks at call sites.
func (s *settingsEmailSender) Configured() bool {
	return s.resolve() != nil
}

func (s *settingsEmailSender) Send(ctx context.Context, recipient, subject, body string) error {
	sender := s.resolve()
	if sender == nil {
		return fmt.Errorf("SMTP sender is not configured")
	}
	return sender.Send(ctx, recipient, subject, body)
}

func (s *smtpEmailSender) Send(ctx context.Context, recipient, subject, body string) error {
	if s == nil {
		return fmt.Errorf("SMTP sender is not configured")
	}
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(s.host, s.port))
	if err != nil {
		return fmt.Errorf("dial SMTP server: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if s.username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.username, s.password, s.host)); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}
	message := fmt.Sprintf("From: PromptOS <%s>\r\nTo: <%s>\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", s.from, recipient, subject, body)
	if _, err := io.WriteString(writer, message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	return client.Quit()
}
