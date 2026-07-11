package mail

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/config"
)

var sensitiveMailPattern = regexp.MustCompile(`(?i)(token=|password=|secret=)[^&\s"'<>]+`)

type LogSender struct {
	fromName string
	from     string
	logBody  bool
}

func NewLogSender(cfg config.Config) *LogSender {
	return &LogSender{
		fromName: strings.TrimSpace(cfg.MailFromName),
		from:     strings.TrimSpace(cfg.MailFrom),
		logBody:  cfg.MailLogBody,
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

	bodyLog := "<redacted>"
	if s.logBody {
		bodyLog = redactSensitiveMailContent(message.Body)
	}

	log.Printf("mail log mode: from=%q to=%q subject=%q body=%s", fromLabel, message.To, message.Subject, bodyLog)
	return nil
}

func redactSensitiveMailContent(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "<redacted>"
	}
	return sensitiveMailPattern.ReplaceAllString(trimmed, "${1}<redacted>")
}
