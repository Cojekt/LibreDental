package services

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"io"
	"math/big"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

func TestBuildEmail(t *testing.T) {
	from := &mail.Address{Name: "Clinique Dentaire Étoile", Address: "rappels@example.com"}
	to := &mail.Address{Address: "patient@example.org"}
	subject := "Rappel : rendez-vous à 14 h"
	body := "Bonjour Jane,\n.Votre rendez-vous est demain.\n" + strings.Repeat("x", 120) + "\nÀ bientôt"
	date := time.Date(2026, 10, 5, 9, 30, 0, 0, time.UTC)

	data, err := buildEmail(from, to, subject, body, "<abc@example.com>", date)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}
	if n, crlf := strings.Count(string(data), "\n"), strings.Count(string(data), "\r\n"); n != crlf {
		t.Errorf("Expected every line to end in CRLF, found %d bare LF", n-crlf)
	}

	msg, err := mail.ReadMessage(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Built message doesn't parse: %v", err)
	}
	gotFrom, err := msg.Header.AddressList("From")
	if err != nil || len(gotFrom) != 1 || gotFrom[0].Name != from.Name || gotFrom[0].Address != from.Address {
		t.Errorf("From = %v, %v; want %v", gotFrom, err, from)
	}
	gotSubject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil || gotSubject != subject {
		t.Errorf("Subject = %q, %v; want %q", gotSubject, err, subject)
	}
	if got, err := msg.Header.Date(); err != nil || !got.Equal(date) {
		t.Errorf("Date = %v, %v; want %v", got, err, date)
	}
	for name, want := range map[string]string{
		"To":                        "<patient@example.org>",
		"Message-ID":                "<abc@example.com>",
		"MIME-Version":              "1.0",
		"Content-Type":              `text/plain; charset="utf-8"`,
		"Content-Transfer-Encoding": "quoted-printable",
	} {
		if got := msg.Header.Get(name); got != want {
			t.Errorf("%s = %q; want %q", name, got, want)
		}
	}
	decoded, err := io.ReadAll(quotedprintable.NewReader(msg.Body))
	if err != nil {
		t.Fatalf("Body doesn't decode: %v", err)
	}
	if want := strings.ReplaceAll(body, "\n", "\r\n"); string(decoded) != want {
		t.Errorf("Body = %q; want %q", decoded, want)
	}

	for _, bad := range []struct {
		from    *mail.Address
		subject string
	}{
		{from, "Reminder\r\nBcc: attacker@example.com"},
		{from, "Reminder\nX-Injected: 1"},
		{&mail.Address{Name: "Clinic\r\nBcc: attacker@example.com", Address: "rappels@example.com"}, "Reminder"},
	} {
		if _, err := buildEmail(bad.from, to, bad.subject, body, "<abc@example.com>", date); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected header injection to be rejected (from %q, subject %q), got %v", bad.from.Name, bad.subject, err)
		}
	}
}

func TestParseSMTPConfig(t *testing.T) {
	cfg, err := parseSMTPConfig(map[string]string{"host": "smtp.example.com", "from_address": "a@example.com"})
	if err != nil {
		t.Fatalf("Expected minimal config to parse, got %v", err)
	}
	if cfg.port != 587 || cfg.tlsMode != smtpTLSModeSTARTTLS || cfg.username != "" {
		t.Errorf("Expected STARTTLS on 587 with no sign-in by default, got %+v", cfg)
	}
	cfg, err = parseSMTPConfig(map[string]string{"host": "smtp.example.com", "from_address": "a@example.com", "tls_mode": "implicit"})
	if err != nil || cfg.port != 465 {
		t.Errorf("Expected implicit TLS to default to port 465, got %+v, %v", cfg, err)
	}
	cfg, err = parseSMTPConfig(map[string]string{
		"host": " email-smtp.us-east-1.amazonaws.com ", "port": "2587", "username": "AKIAEXAMPLE", "password": "secret",
		"from_address": "reminders@clinic.example", "from_name": "Smile Dental",
	})
	if err != nil || cfg.host != "email-smtp.us-east-1.amazonaws.com" || cfg.port != 2587 || cfg.from.String() != `"Smile Dental" <reminders@clinic.example>` {
		t.Errorf("Unexpected full config: %+v, %v", cfg, err)
	}

	_, err = parseSMTPConfig(map[string]string{"port": "99999", "tls_mode": "none", "username": "user", "from_address": "Clinic <a@example.com>"})
	if !errors.Is(err, storage.ErrInvalidInput) {
		t.Fatalf("Expected invalid config to be rejected, got %v", err)
	}
	for _, want := range []string{"host is required", "TLS mode", "port must be", "password is required", "not a valid email address"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected all problems reported at once; %q missing from %q", want, err)
		}
	}
	if _, err := parseSMTPConfig(map[string]string{"host": "h", "from_address": "a@example.com", "password": "p"}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected a password without a username to be rejected, got %v", err)
	}
}

// fakeSMTPServer is a minimal SMTP server for exercising the provider's TLS, sign-in, and
// failure handling without a network.
type fakeSMTPServer struct {
	ln          net.Listener
	tlsConfig   *tls.Config
	implicitTLS bool
	// offerSTARTTLS controls whether EHLO advertises STARTTLS on a plain connection.
	offerSTARTTLS bool
	// authMechanisms is advertised after EHLO; empty means no AUTH extension.
	authMechanisms string
	rejectAuth     bool
	rejectRcpt     bool
	// afterData is the reply to the end of the message: "ok", "reject", "drop", or "hang".
	afterData string
	done      chan struct{}

	mu          sync.Mutex
	authUser    string
	authPass    string
	authOverTLS bool
	data        string
}

func newFakeSMTPServer(t *testing.T, cert tls.Certificate) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := &fakeSMTPServer{
		ln:             ln,
		tlsConfig:      &tls.Config{Certificates: []tls.Certificate{cert}},
		offerSTARTTLS:  true,
		authMechanisms: "PLAIN LOGIN",
		afterData:      "ok",
		done:           make(chan struct{}),
	}
	t.Cleanup(func() {
		close(s.done)
		ln.Close()
	})
	return s
}

func (s *fakeSMTPServer) start() {
	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
}

func (s *fakeSMTPServer) port() string {
	return strconv.Itoa(s.ln.Addr().(*net.TCPAddr).Port)
}

func (s *fakeSMTPServer) serve(conn net.Conn) {
	defer conn.Close()
	isTLS := s.implicitTLS
	if s.implicitTLS {
		conn = tls.Server(conn, s.tlsConfig)
	}
	tp := textproto.NewConn(conn)
	reply := func(lines ...string) {
		for _, l := range lines {
			if err := tp.PrintfLine("%s", l); err != nil {
				return
			}
		}
	}
	reply("220 fake.example ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "EHLO", "HELO":
			lines := []string{"250-fake.example"}
			if s.offerSTARTTLS && !isTLS {
				lines = append(lines, "250-STARTTLS")
			}
			if s.authMechanisms != "" {
				lines = append(lines, "250-AUTH "+s.authMechanisms)
			}
			reply(append(lines, "250 8BITMIME")...)
		case "STARTTLS":
			reply("220 ready to start TLS")
			tlsConn := tls.Server(conn, s.tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn, isTLS = tlsConn, true
			tp = textproto.NewConn(conn)
		case "AUTH":
			mech, resp, _ := strings.Cut(arg, " ")
			decoded, _ := base64.StdEncoding.DecodeString(resp)
			parts := strings.Split(string(decoded), "\x00")
			if strings.ToUpper(mech) != "PLAIN" || len(parts) != 3 {
				reply("504 unsupported")
				continue
			}
			s.mu.Lock()
			s.authUser, s.authPass, s.authOverTLS = parts[1], parts[2], isTLS
			s.mu.Unlock()
			if s.rejectAuth {
				reply("535 5.7.8 authentication failed")
			} else {
				reply("235 2.7.0 authenticated")
			}
		case "MAIL":
			reply("250 2.1.0 ok")
		case "RCPT":
			if s.rejectRcpt {
				reply("550 5.1.1 no such user")
			} else {
				reply("250 2.1.5 ok")
			}
		case "DATA":
			reply("354 end with .")
			data, err := tp.ReadDotBytes()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.data = string(data)
			s.mu.Unlock()
			switch s.afterData {
			case "reject":
				reply("554 5.6.0 message rejected")
			case "drop":
				return
			case "hang":
				<-s.done
				return
			default:
				reply("250 2.0.0 Ok: queued")
			}
		case "QUIT":
			reply("221 2.0.0 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

func (s *fakeSMTPServer) received() (user, pass string, overTLS bool, data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.authUser, s.authPass, s.authOverTLS, s.data
}

// newTestCertificate returns a self-signed certificate for 127.0.0.1 and a pool that trusts it.
func newTestCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fake smtp"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, pool
}

func TestSMTPEmailProvider_Send(t *testing.T) {
	cert, pool := newTestCertificate(t)
	msg := &domain.NotificationMessage{
		Channel: domain.NotificationChannelEmail,
		To:      "patient@example.org",
		Subject: "Appointment reminder",
		Body:    "See you tomorrow at 2:00 PM.",
	}

	tests := []struct {
		name       string
		setup      func(s *fakeSMTPServer, config map[string]string)
		untrusted  bool
		wantErr    string
		wantUnsure bool
		// wantNoAuth means credentials must never have reached the server.
		wantNoAuth bool
		wantData   bool
	}{
		{name: "STARTTLS", wantData: true},
		{name: "implicit TLS", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.implicitTLS = true
			c["tls_mode"] = "implicit"
		}, wantData: true},
		{name: "no sign-in configured", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.authMechanisms = ""
			delete(c, "username")
			delete(c, "password")
		}, wantData: true, wantNoAuth: true},
		{name: "STARTTLS not offered", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.offerSTARTTLS = false
		}, wantErr: "does not offer STARTTLS", wantNoAuth: true},
		{name: "untrusted certificate", untrusted: true, wantErr: "could not start TLS", wantNoAuth: true},
		{name: "AUTH PLAIN not offered", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.authMechanisms = "LOGIN XOAUTH2"
		}, wantErr: "AUTH PLAIN", wantNoAuth: true},
		{name: "sign-in rejected", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.rejectAuth = true
		}, wantErr: "sign-in failed"},
		{name: "recipient rejected", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.rejectRcpt = true
		}, wantErr: "refused recipient"},
		{name: "message rejected", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.afterData = "reject"
		}, wantErr: "rejected message", wantData: true},
		{name: "connection dropped after message", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.afterData = "drop"
		}, wantErr: "no reply", wantUnsure: true, wantData: true},
		{name: "no reply after message", setup: func(s *fakeSMTPServer, c map[string]string) {
			s.afterData = "hang"
		}, wantErr: "no reply", wantUnsure: true, wantData: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeSMTPServer(t, cert)
			config := map[string]string{
				"host": "127.0.0.1", "username": "smtp-user", "password": "smtp-pass",
				"from_address": "reminders@clinic.example", "from_name": "Smile Dental",
			}
			if tt.setup != nil {
				tt.setup(server, config)
			}
			config["port"] = server.port()
			server.start()

			provider := NewSMTPEmailProvider()
			if !tt.untrusted {
				provider.rootCAs = pool
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			start := time.Now()
			result, err := provider.Send(ctx, msg, config)
			if time.Since(start) > 3*time.Second {
				t.Errorf("Send took %v; the context deadline should bound it", time.Since(start))
			}

			user, pass, overTLS, data := server.received()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Send failed: %v", err)
				}
				if result.Status != domain.NotificationStatusSent || result.ExternalMessageID == "" {
					t.Errorf("Unexpected result: %+v", result)
				}
				if !strings.Contains(data, "Message-ID: "+result.ExternalMessageID) {
					t.Errorf("Expected the recorded ID %q to be the message's Message-ID", result.ExternalMessageID)
				}
				if !strings.Contains(data, "Subject: Appointment reminder") || !strings.Contains(data, "See you tomorrow at 2:00 PM.") {
					t.Errorf("Message didn't arrive intact:\n%s", data)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Expected error containing %q, got %v", tt.wantErr, err)
				}
				if got := errors.Is(err, domain.ErrDeliveryUnknown); got != tt.wantUnsure {
					t.Errorf("errors.Is(err, ErrDeliveryUnknown) = %v; want %v (err: %v)", got, tt.wantUnsure, err)
				}
				if tt.wantUnsure && (result == nil || result.ExternalMessageID == "") {
					t.Errorf("Expected an uncertain send to keep its message ID for matching bounces, got %+v", result)
				}
			}

			if tt.wantNoAuth {
				if user != "" || pass != "" {
					t.Errorf("Credentials reached the server: %q/%q", user, pass)
				}
			} else if user != "smtp-user" || pass != "smtp-pass" || !overTLS {
				t.Errorf("Expected credentials sent only over TLS, got %q/%q overTLS=%v", user, pass, overTLS)
			}
			if got := data != ""; got != tt.wantData {
				t.Errorf("Message data received = %v; want %v", got, tt.wantData)
			}
		})
	}
}

func TestSMTPEmailProvider_SendFailsBeforeConnecting(t *testing.T) {
	provider := NewSMTPEmailProvider()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	config := map[string]string{"host": "127.0.0.1", "port": strconv.Itoa(port), "from_address": "reminders@clinic.example"}
	for _, to := range []string{"not an address", "a@example.com\r\nBcc: b@example.com"} {
		msg := &domain.NotificationMessage{To: to, Subject: "s", Body: "b"}
		if _, err := provider.Send(ctx, msg, config); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected recipient %q to be rejected, got %v", to, err)
		}
	}

	if _, err := provider.Send(ctx, &domain.NotificationMessage{To: "a@example.com", Body: "b"}, map[string]string{}); !errors.Is(err, storage.ErrInvalidInput) {
		t.Errorf("Expected missing settings to be rejected, got %v", err)
	}

	// Nothing listens on port 1, so this is a connection failure: definitely not sent.
	_, err := provider.Send(ctx, &domain.NotificationMessage{To: "a@example.com", Subject: "s", Body: "b"}, config)
	if err == nil || errors.Is(err, domain.ErrDeliveryUnknown) {
		t.Errorf("Expected a connection failure that is not uncertain, got %v", err)
	}
}

// TestSMTPLive sends a real email. It only runs when SMTP_LIVE_HOST is set; with Amazon SES,
// the default recipient is SES's mailbox simulator, which accepts mail without delivering it.
//
//	SMTP_LIVE_HOST=email-smtp.us-east-1.amazonaws.com SMTP_LIVE_USERNAME=... SMTP_LIVE_PASSWORD=... \
//	SMTP_LIVE_FROM=reminders@your-verified-domain go test ./internal/services -run TestSMTPLive -v
func TestSMTPLive(t *testing.T) {
	host := os.Getenv("SMTP_LIVE_HOST")
	if host == "" {
		t.Skip("SMTP_LIVE_HOST not set")
	}
	to := os.Getenv("SMTP_LIVE_TO")
	if to == "" {
		to = "success@simulator.amazonses.com"
	}
	config := map[string]string{
		"host":         host,
		"port":         os.Getenv("SMTP_LIVE_PORT"),
		"tls_mode":     os.Getenv("SMTP_LIVE_TLS_MODE"),
		"username":     os.Getenv("SMTP_LIVE_USERNAME"),
		"password":     os.Getenv("SMTP_LIVE_PASSWORD"),
		"from_address": os.Getenv("SMTP_LIVE_FROM"),
		"from_name":    "LibreDental live test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := NewSMTPEmailProvider().Send(ctx, &domain.NotificationMessage{
		To: to, Subject: "LibreDental SMTP live test", Body: "This is a test message from LibreDental's test suite.",
	}, config)
	if err != nil {
		t.Fatalf("Live send failed: %v", err)
	}
	t.Logf("Sent; Message-ID %s", result.ExternalMessageID)
}
