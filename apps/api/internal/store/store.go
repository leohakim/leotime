package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/auth"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrSessionNotFound = errors.New("session not found")

type Store struct {
	db *sql.DB
}

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Locale     string `json:"locale"`
	LayoutMode string `json:"layoutMode"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type Overview struct {
	ClientsTotal     int `json:"clientsTotal"`
	ProjectsTotal    int `json:"projectsTotal"`
	TasksTotal       int `json:"tasksTotal"`
	TagsTotal        int `json:"tagsTotal"`
	TimeEntriesTotal int `json:"timeEntriesTotal"`
	InvoicesTotal    int `json:"invoicesTotal"`
	OpenTimers       int `json:"openTimers"`
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) BootstrapAdmin(ctx context.Context, email string, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return errors.New("bootstrap email is required")
	}
	if password == "" {
		return errors.New("bootstrap password is required")
	}

	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap password: %w", err)
	}

	userID, err := newID("usr")
	if err != nil {
		return err
	}
	now := nowString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap admin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, locale, layout_mode, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'es', 'solid', ?, ?)
	`, userID, email, "Administrador", passwordHash, now, now); err != nil {
		return fmt.Errorf("insert bootstrap user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO app_settings (user_id, updated_at)
		VALUES (?, ?)
	`, userID, now); err != nil {
		return fmt.Errorf("insert bootstrap settings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap admin: %w", err)
	}

	return nil
}

func (s *Store) Authenticate(ctx context.Context, email string, password string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	var user User
	var passwordHash string
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, locale, layout_mode, created_at, updated_at
		FROM users
		WHERE email = ?
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&passwordHash,
		&user.Locale,
		&user.LayoutMode,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	if !auth.VerifyPassword(passwordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

func (s *Store) CreateSession(ctx context.Context, userID string, ttl time.Duration) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().UTC().Add(ttl)
	now := nowString()
	tokenHash := hashToken(token)

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, tokenHash, userID, expiresAt.Format(time.RFC3339Nano), now); err != nil {
		return "", time.Time{}, fmt.Errorf("insert session: %w", err)
	}

	return token, expiresAt, nil
}

func (s *Store) UserBySessionToken(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	var user User
	if err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.locale, u.layout_mode, u.created_at, u.updated_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ?
			AND s.expires_at > ?
	`, hashToken(token), nowString()).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Locale,
		&user.LayoutMode,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("query session user: %w", err)
	}

	return &user, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", hashToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Store) Overview(ctx context.Context, userID string) (Overview, error) {
	counts := []struct {
		query string
		into  *int
	}{}

	overview := Overview{}
	counts = append(counts,
		struct {
			query string
			into  *int
		}{"SELECT COUNT(*) FROM clients WHERE user_id = ? AND archived_at IS NULL", &overview.ClientsTotal},
		struct {
			query string
			into  *int
		}{"SELECT COUNT(*) FROM projects WHERE user_id = ? AND archived_at IS NULL", &overview.ProjectsTotal},
		struct {
			query string
			into  *int
		}{"SELECT COUNT(*) FROM tasks WHERE user_id = ? AND archived_at IS NULL", &overview.TasksTotal},
		struct {
			query string
			into  *int
		}{"SELECT COUNT(*) FROM tags WHERE user_id = ? AND archived_at IS NULL", &overview.TagsTotal},
		struct {
			query string
			into  *int
		}{"SELECT COUNT(*) FROM time_entries WHERE user_id = ?", &overview.TimeEntriesTotal},
		struct {
			query string
			into  *int
		}{"SELECT COUNT(*) FROM invoices WHERE user_id = ?", &overview.InvoicesTotal},
		struct {
			query string
			into  *int
		}{"SELECT COUNT(*) FROM time_entries WHERE user_id = ? AND ended_at IS NULL", &overview.OpenTimers},
	)

	for _, count := range counts {
		if err := s.db.QueryRowContext(ctx, count.query, userID).Scan(count.into); err != nil {
			return Overview{}, fmt.Errorf("run overview count: %w", err)
		}
	}

	return overview, nil
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
