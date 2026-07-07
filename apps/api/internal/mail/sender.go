package mail

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/config"
)

var ErrInvalidMessage = errors.New("invalid mail message")

type Message struct {
	To      string
	Subject string
	Body    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

func NewSender(cfg config.Config) (Sender, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.MailMode)) {
	case "log":
		return NewLogSender(cfg), nil
	case "smtp":
		return NewSMTPSender(cfg)
	default:
		return nil, fmt.Errorf("unsupported mail mode %q", cfg.MailMode)
	}
}

func ValidateMessage(message Message) error {
	if strings.TrimSpace(message.To) == "" {
		return fmt.Errorf("%w: recipient is required", ErrInvalidMessage)
	}
	if strings.TrimSpace(message.Subject) == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidMessage)
	}
	if strings.TrimSpace(message.Body) == "" {
		return fmt.Errorf("%w: body is required", ErrInvalidMessage)
	}
	return nil
}
