package mail

import (
	"context"
	"errors"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/config"
)

func TestLogSenderSend(t *testing.T) {
	sender := NewLogSender(config.Config{
		MailFrom:     "no-reply@localhost",
		MailFromName: "leotime",
	})

	if err := sender.Send(context.Background(), Message{
		To:      "admin@example.com",
		Subject: "Test",
		Body:    "Hello",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestValidateMessageRequiresFields(t *testing.T) {
	if err := ValidateMessage(Message{}); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected invalid message, got %v", err)
	}
}

func TestNewSMTPSenderRequiresHost(t *testing.T) {
	if _, err := NewSMTPSender(config.Config{MailFrom: "no-reply@localhost"}); err == nil {
		t.Fatal("expected smtp host validation error")
	}
}

func TestIsPermanent(t *testing.T) {
	if !IsPermanent(Permanent(errors.New("550 rejected"))) {
		t.Fatal("expected permanent error")
	}
	if IsPermanent(Transient(errors.New("timeout"))) {
		t.Fatal("expected transient error")
	}
}

func TestNewSenderSupportsLogMode(t *testing.T) {
	sender, err := NewSender(config.Config{MailMode: "log"})
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}
	if _, ok := sender.(*LogSender); !ok {
		t.Fatalf("expected log sender, got %T", sender)
	}
}
