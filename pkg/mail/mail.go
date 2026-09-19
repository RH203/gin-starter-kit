package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"
	"path"
	"strings"

	"gin-starter-pack/templates"
)

// Config holds email transport and authentication settings
type Config struct {
	Driver      string // "smtp" or "log"
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	Encryption  string // "tls", "ssl", or "none"
}

// Message encapsulates email recipients, headers, and body
type Message struct {
	To      []string
	Subject string
	Body    string
	IsHTML  bool
}

// Mailer defines the contract for sending emails
type Mailer interface {
	Send(ctx context.Context, msg *Message) error
	SendSimple(ctx context.Context, to, subject, body string, isHTML bool) error
	SendTemplate(ctx context.Context, to, subject, templateName string, data interface{}) error
}

// RenderTemplate renders an HTML email template stored in templates/emails/
func RenderTemplate(templateName string, data interface{}) (string, error) {
	cleanName := path.Base(templateName)
	tmplPath := "emails/" + cleanName

	tmpl, err := template.ParseFS(templates.EmailFS, tmplPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template %s: %w", cleanName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template %s: %w", cleanName, err)
	}

	return buf.String(), nil
}

// NewMailer returns an appropriate Mailer implementation based on configuration
func NewMailer(cfg *Config) Mailer {
	if cfg == nil || strings.ToLower(cfg.Driver) == "log" || cfg.Host == "" {
		return NewLogMailer(cfg)
	}
	return NewSMTPMailer(cfg)
}

// LogMailer logs outgoing emails without sending to a remote server
type LogMailer struct {
	cfg *Config
}

// NewLogMailer creates a logger-based mailer for development
func NewLogMailer(cfg *Config) *LogMailer {
	return &LogMailer{cfg: cfg}
}

func (m *LogMailer) Send(ctx context.Context, msg *Message) error {
	from := "noreply@example.com"
	if m.cfg != nil && m.cfg.FromAddress != "" {
		from = m.cfg.FromAddress
	}
	slog.Info("Email dispatched via log mailer",
		"from", from,
		"to", strings.Join(msg.To, ", "),
		"subject", msg.Subject,
		"is_html", msg.IsHTML,
	)
	return nil
}

func (m *LogMailer) SendSimple(ctx context.Context, to, subject, body string, isHTML bool) error {
	return m.Send(ctx, &Message{
		To:      []string{to},
		Subject: subject,
		Body:    body,
		IsHTML:  isHTML,
	})
}

func (m *LogMailer) SendTemplate(ctx context.Context, to, subject, templateName string, data interface{}) error {
	rendered, err := RenderTemplate(templateName, data)
	if err != nil {
		return err
	}
	return m.Send(ctx, &Message{
		To:      []string{to},
		Subject: subject,
		Body:    rendered,
		IsHTML:  true,
	})
}

// SMTPMailer sends emails via SMTP protocol
type SMTPMailer struct {
	cfg *Config
}

// NewSMTPMailer creates an SMTP mailer instance
func NewSMTPMailer(cfg *Config) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) Send(ctx context.Context, msg *Message) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	fromHeader := m.cfg.FromAddress
	if m.cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromAddress)
	}

	contentType := "text/plain; charset=UTF-8"
	if msg.IsHTML {
		contentType = "text/html; charset=UTF-8"
	}

	headers := make(map[string]string)
	headers["From"] = fromHeader
	headers["To"] = strings.Join(msg.To, ", ")
	headers["Subject"] = msg.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = contentType

	var messageBuilder strings.Builder
	for k, v := range headers {
		messageBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	messageBuilder.WriteString("\r\n")
	messageBuilder.WriteString(msg.Body)

	rawMessage := []byte(messageBuilder.String())

	var auth smtp.Auth
	if m.cfg.Username != "" && m.cfg.Password != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	// Handle direct SSL connection
	if m.cfg.Encryption == "ssl" || m.cfg.Port == 465 {
		tlsConfig := &tls.Config{
			ServerName: m.cfg.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP SSL: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, m.cfg.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if auth != nil {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("failed to authenticate SMTP: %w", err)
			}
		}

		if err = client.Mail(m.cfg.FromAddress); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}
		for _, recipient := range msg.To {
			if err = client.Rcpt(recipient); err != nil {
				return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
			}
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to open data writer: %w", err)
		}
		if _, err = w.Write(rawMessage); err != nil {
			return fmt.Errorf("failed to write message body: %w", err)
		}
		if err = w.Close(); err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}
		return client.Quit()
	}

	// Standard SMTP with opportunistic STARTTLS
	return smtp.SendMail(addr, auth, m.cfg.FromAddress, msg.To, rawMessage)
}

func (m *SMTPMailer) SendSimple(ctx context.Context, to, subject, body string, isHTML bool) error {
	return m.Send(ctx, &Message{
		To:      []string{to},
		Subject: subject,
		Body:    body,
		IsHTML:  isHTML,
	})
}

func (m *SMTPMailer) SendTemplate(ctx context.Context, to, subject, templateName string, data interface{}) error {
	rendered, err := RenderTemplate(templateName, data)
	if err != nil {
		return err
	}
	return m.Send(ctx, &Message{
		To:      []string{to},
		Subject: subject,
		Body:    rendered,
		IsHTML:  true,
	})
}
