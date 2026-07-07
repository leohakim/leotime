package mail

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/config"
)

type LogSender struct {
	fromName string
	from     string
}

func NewLogSender(cfg config.Config) *LogSender {
	return &LogSender{
		fromName: strings.TrimSpace(cfg.MailFromName),
		from:     strings.TrimSpace(cfg.MailFrom),
	}
}

func (s *LogSender) Send(_ context.Context, message Message) error {
	if err := ValidateMessage(message); err != nil {
		return Permanent(err)
	}

	fromLabel := s.from
	if s.fromName != "" {
		fromLabel = fmt.Sprintf("%s <%s>", s.fromName, s.from)
	}

	log.Printf("mail log mode: from=%q to=%q subject=%q body=%q", fromLabel, message.To, message.Subject, message.Body)
	return nil
}
