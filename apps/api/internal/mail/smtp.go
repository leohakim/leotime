package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
)

type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
	useTLS   bool
}

func NewSMTPSender(cfg config.Config) (*SMTPSender, error) {
	host := strings.TrimSpace(cfg.SMTPHost)
	if host == "" {
		return nil, fmt.Errorf("LEOTIME_SMTP_HOST is required when LEOTIME_MAIL_MODE=smtp")
	}
	from := strings.TrimSpace(cfg.MailFrom)
	if from == "" {
		return nil, fmt.Errorf("LEOTIME_MAIL_FROM is required when LEOTIME_MAIL_MODE=smtp")
	}

	return &SMTPSender{
		host:     host,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     from,
		fromName: strings.TrimSpace(cfg.MailFromName),
		useTLS:   cfg.SMTPTLS,
	}, nil
}

func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	if err := ValidateMessage(message); err != nil {
		return Permanent(err)
	}
	if _, err := mail.ParseAddress(message.To); err != nil {
		return Permanent(fmt.Errorf("invalid recipient: %w", err))
	}

	type result struct {
		err error
	}
	done := make(chan result, 1)
	go func() {
		done <- result{err: s.send(message)}
	}()

	select {
	case <-ctx.Done():
		return Transient(ctx.Err())
	case res := <-done:
		return res.err
	}
}

func (s *SMTPSender) send(message Message) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	body := buildPlainTextBody(s.from, s.fromName, message.To, message.Subject, message.Body)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	if s.port == 465 {
		return s.sendImplicitTLS(addr, auth, message.To, body)
	}

	if err := smtp.SendMail(addr, auth, s.from, []string{message.To}, body); err != nil {
		return classifySMTPError(err)
	}
	return nil
}

func (s *SMTPSender) sendImplicitTLS(addr string, auth smtp.Auth, to string, body []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp", addr, &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return Transient(err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return classifySMTPError(err)
	}
	defer client.Close()

	return s.deliver(client, auth, to, body)
}

func (s *SMTPSender) deliver(client *smtp.Client, auth smtp.Auth, to string, body []byte) error {
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return classifySMTPError(err)
			}
		}
	}
	if err := client.Mail(s.from); err != nil {
		return classifySMTPError(err)
	}
	if err := client.Rcpt(to); err != nil {
		return classifySMTPError(err)
	}
	writer, err := client.Data()
	if err != nil {
		return classifySMTPError(err)
	}
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return classifySMTPError(err)
	}
	if err := writer.Close(); err != nil {
		return classifySMTPError(err)
	}
	return client.Quit()
}

func buildPlainTextBody(from string, fromName string, to string, subject string, body string) []byte {
	fromHeader := from
	if fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", fromName, from)
	}

	var builder strings.Builder
	builder.WriteString("From: ")
	builder.WriteString(fromHeader)
	builder.WriteString("\r\nTo: ")
	builder.WriteString(to)
	builder.WriteString("\r\nSubject: ")
	builder.WriteString(subject)
	builder.WriteString("\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	builder.WriteString(body)
	builder.WriteString("\r\n")
	return []byte(builder.String())
}

func classifySMTPError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "authentication"),
		strings.Contains(message, "535"),
		strings.Contains(message, "534"),
		strings.Contains(message, "550"),
		strings.Contains(message, "553"),
		strings.Contains(message, "501"),
		strings.Contains(message, "552"):
		return Permanent(err)
	default:
		return Transient(err)
	}
}
