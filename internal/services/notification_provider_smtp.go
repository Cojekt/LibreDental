package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

const (
	smtpTLSModeSTARTTLS = "starttls"
	smtpTLSModeImplicit = "implicit"
)

// SMTPEmailProvider sends email through any SMTP server that supports TLS and AUTH PLAIN, such
// as Amazon SES, Google Workspace (with an app password), or a practice's own mail server.
//
// TLS is required for the whole session, never just offered: smtp.SendMail only upgrades when
// the server advertises STARTTLS, so a man-in-the-middle could strip it and read the message.
// The connection is dialed here instead, which also lets the context bound the conversation,
// since net/smtp has no context support.
type SMTPEmailProvider struct {
	// rootCAs replaces the system certificate pool; only tests set it.
	rootCAs *x509.CertPool
}

func NewSMTPEmailProvider() *SMTPEmailProvider {
	return &SMTPEmailProvider{}
}

func (p *SMTPEmailProvider) Name() string { return "smtp_email" }

func (p *SMTPEmailProvider) Channel() domain.NotificationChannel {
	return domain.NotificationChannelEmail
}

type smtpConfig struct {
	host     string
	port     int
	tlsMode  string
	username string
	password string
	from     *mail.Address
}

// parseSMTPConfig validates the stored settings and reports every problem at once.
func parseSMTPConfig(config map[string]string) (*smtpConfig, error) {
	var problems []string
	cfg := &smtpConfig{
		host:     strings.TrimSpace(config["host"]),
		tlsMode:  strings.TrimSpace(config["tls_mode"]),
		username: strings.TrimSpace(config["username"]),
		password: config["password"],
	}

	if cfg.host == "" {
		problems = append(problems, "host is required")
	}
	switch cfg.tlsMode {
	case "":
		cfg.tlsMode = smtpTLSModeSTARTTLS
	case smtpTLSModeSTARTTLS, smtpTLSModeImplicit:
	default:
		problems = append(problems, fmt.Sprintf("TLS mode must be %q or %q", smtpTLSModeSTARTTLS, smtpTLSModeImplicit))
	}

	switch port := strings.TrimSpace(config["port"]); port {
	case "":
		cfg.port = 587
		if cfg.tlsMode == smtpTLSModeImplicit {
			cfg.port = 465
		}
	default:
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			problems = append(problems, "port must be a number from 1 to 65535")
		}
		cfg.port = n
	}

	if cfg.username != "" && cfg.password == "" {
		problems = append(problems, "password is required when a username is set")
	}
	if cfg.username == "" && cfg.password != "" {
		problems = append(problems, "username is required when a password is set")
	}

	fromAddress := strings.TrimSpace(config["from_address"])
	fromName := strings.TrimSpace(config["from_name"])
	switch {
	case fromAddress == "":
		problems = append(problems, "from address is required")
	case strings.ContainsAny(fromAddress+fromName, "\r\n"):
		problems = append(problems, "from address and name must be a single line")
	default:
		addr, err := mail.ParseAddress(fromAddress)
		if err != nil || addr.Name != "" {
			problems = append(problems, fmt.Sprintf("from address %q is not a valid email address", fromAddress))
		} else {
			cfg.from = &mail.Address{Name: fromName, Address: addr.Address}
		}
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("%w: SMTP settings: %s", storage.ErrInvalidInput, strings.Join(problems, "; "))
	}
	return cfg, nil
}

func (p *SMTPEmailProvider) Send(ctx context.Context, msg *domain.NotificationMessage, config map[string]string) (*domain.NotificationResult, error) {
	cfg, err := parseSMTPConfig(config)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(msg.To, "\r\n") {
		return nil, fmt.Errorf("%w: recipient must be a single line", storage.ErrInvalidInput)
	}
	to, err := mail.ParseAddress(msg.To)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not a valid email address", storage.ErrInvalidInput, msg.To)
	}

	messageID, err := newMessageID(cfg.from.Address)
	if err != nil {
		return nil, err
	}
	data, err := buildEmail(cfg.from, to, msg.Subject, msg.Body, messageID, time.Now())
	if err != nil {
		return nil, err
	}

	// net/smtp can't return the server's own message ID, so ours (also quoted in bounces) is
	// what the log records, even for a failed or uncertain send.
	result := &domain.NotificationResult{ExternalMessageID: messageID, Status: domain.NotificationStatusFailed}
	if err := p.deliver(ctx, cfg, to.Address, data); err != nil {
		return result, err
	}
	result.Status = domain.NotificationStatusSent
	return result, nil
}

func (p *SMTPEmailProvider) deliver(ctx context.Context, cfg *smtpConfig, to string, data []byte) error {
	addr := net.JoinHostPort(cfg.host, strconv.Itoa(cfg.port))
	tlsConfig := &tls.Config{ServerName: cfg.host, MinVersion: tls.VersionTLS12, RootCAs: p.rootCAs}

	var conn net.Conn
	var err error
	if cfg.tlsMode == smtpTLSModeImplicit {
		conn, err = (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("could not connect to SMTP server %s: %w", addr, err)
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	c, err := smtp.NewClient(conn, cfg.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP server %s did not respond: %w", addr, err)
	}
	defer c.Close()

	if cfg.tlsMode == smtpTLSModeSTARTTLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server %s does not offer STARTTLS; refusing to send without encryption", addr)
		}
		if err := c.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("could not start TLS with SMTP server %s: %w", addr, err)
		}
	}

	if cfg.username != "" {
		ok, mechanisms := c.Extension("AUTH")
		if !ok || !slices.Contains(strings.Fields(strings.ToUpper(mechanisms)), "PLAIN") {
			return fmt.Errorf("SMTP server %s does not support password sign-in (AUTH PLAIN)", addr)
		}
		if err := c.Auth(smtp.PlainAuth("", cfg.username, cfg.password, cfg.host)); err != nil {
			return fmt.Errorf("SMTP sign-in failed: %w", err)
		}
	}

	if err := c.Mail(cfg.from.Address); err != nil {
		return fmt.Errorf("SMTP server refused sender %s: %w", cfg.from.Address, err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP server refused recipient: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP server refused message: %w", err)
	}
	// A message cut off mid-way is discarded by the server, so a write failure means not sent.
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("sending message to SMTP server failed: %w", err)
	}
	// Close ends the message and reads the server's verdict. A rejection means not sent; no
	// reply at all means the server may have accepted it.
	if err := w.Close(); err != nil {
		var smtpErr *textproto.Error
		if errors.As(err, &smtpErr) {
			return fmt.Errorf("SMTP server rejected message: %w", err)
		}
		return fmt.Errorf("%w: no reply from SMTP server after sending message: %v", domain.ErrDeliveryUnknown, err)
	}
	// The message is accepted at this point; a failed QUIT doesn't change that.
	_ = c.Quit()
	return nil
}

// buildEmail builds a plain-text RFC 5322 message. Header values containing CR or LF are
// rejected outright, so nothing in a subject or name can inject extra headers.
func buildEmail(from, to *mail.Address, subject, body, messageID string, date time.Time) ([]byte, error) {
	for _, v := range []string{from.Name, from.Address, to.Name, to.Address, subject, messageID} {
		if strings.ContainsAny(v, "\r\n") {
			return nil, fmt.Errorf("%w: email headers must be a single line", storage.ErrInvalidInput)
		}
	}

	var b bytes.Buffer
	header := func(name, value string) {
		b.WriteString(name + ": " + value + "\r\n")
	}
	header("From", from.String())
	header("To", to.String())
	header("Subject", mime.QEncoding.Encode("utf-8", subject))
	header("Date", date.Format(time.RFC1123Z))
	header("Message-ID", messageID)
	header("MIME-Version", "1.0")
	header("Content-Type", `text/plain; charset="utf-8"`)
	header("Content-Transfer-Encoding", "quoted-printable")
	b.WriteString("\r\n")

	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// newMessageID returns a random Message-ID at the sender's domain.
func newMessageID(fromAddress string) (string, error) {
	domainPart := fromAddress[strings.LastIndex(fromAddress, "@")+1:]
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate message ID: %w", err)
	}
	return "<" + hex.EncodeToString(buf) + "@" + domainPart + ">", nil
}
