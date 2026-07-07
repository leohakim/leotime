package notify

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type PasswordResetService struct {
	store  *store.Store
	outbox *outbox.Store
	cfg    config.Config
}

func NewPasswordResetService(st *store.Store, outboxStore *outbox.Store, cfg config.Config) *PasswordResetService {
	return &PasswordResetService{
		store:  st,
		outbox: outboxStore,
		cfg:    cfg,
	}
}

func (s *PasswordResetService) RequestReset(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil
	}

	user, err := s.store.UserByEmail(ctx, email)
	if errors.Is(err, store.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup password reset user: %w", err)
	}

	rawToken, err := s.store.CreatePasswordResetToken(ctx, user.ID, s.cfg.PasswordResetTTL)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}

	resetURL := passwordResetURL(s.cfg.PublicBaseURL, rawToken)
	_, err = s.outbox.Enqueue(ctx, outbox.EnqueueInput{
		UserID:      user.ID,
		Kind:        outbox.KindPasswordReset,
		ToAddress:   user.Email,
		Subject:     passwordResetSubject(user.Locale),
		BodyText:    passwordResetBody(user, resetURL, s.cfg.PasswordResetTTL),
		MaxAttempts: s.cfg.MailMaxAttempts,
	}, time.Now())
	if err != nil {
		return fmt.Errorf("enqueue password reset email: %w", err)
	}
	return nil
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	return s.store.ResetPasswordWithToken(ctx, token, newPassword)
}

func passwordResetURL(publicBaseURL string, rawToken string) string {
	base := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	return fmt.Sprintf("%s#reset-password?token=%s", base, url.QueryEscape(rawToken))
}

func passwordResetSubject(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "Reset your leotime password"
	}
	return "Restablece tu contrasena de leotime"
}

func passwordResetBody(user *store.User, resetURL string, ttl time.Duration) string {
	hours := int(ttl.Hours())
	if hours <= 0 {
		hours = 1
	}

	if strings.EqualFold(strings.TrimSpace(user.Locale), "en") {
		return fmt.Sprintf(
			"Hi %s,\n\nWe received a request to reset your leotime password.\n\nOpen this link to choose a new password:\n%s\n\nThis link expires in about %d hour(s). If you did not request this, you can ignore this email.\n",
			displayName(user.Name, user.Email),
			resetURL,
			hours,
		)
	}

	return fmt.Sprintf(
		"Hola %s,\n\nRecibimos una solicitud para restablecer tu contrasena de leotime.\n\nAbre este enlace para elegir una nueva contrasena:\n%s\n\nEl enlace caduca en aproximadamente %d hora(s). Si no solicitaste esto, puedes ignorar este correo.\n",
		displayName(user.Name, user.Email),
		resetURL,
		hours,
	)
}
