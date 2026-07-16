package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func runResetDataCommand(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("leotime reset-data", flag.ContinueOnError)
	userEmail := flags.String("user-email", "", "owner email (defaults to LEOTIME_BOOTSTRAP_EMAIL)")
	if err := flags.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	if err := db.Migrate(ctx, database); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, cfg.BootstrapEmail, cfg.BootstrapPassword); err != nil {
		return fmt.Errorf("bootstrap admin: %w", err)
	}

	email := cfg.BootstrapEmail
	if *userEmail != "" {
		email = *userEmail
	}
	user, err := st.UserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("load reset user %q: %w", email, err)
	}

	summary, err := st.ClearUserData(ctx, user.ID)
	if err != nil {
		return err
	}

	removedFiles, removeErrors := removeDocumentFiles(cfg.DocumentRoot, summary.DocumentPaths)
	summary.DocumentPaths = removedFiles
	if len(removeErrors) > 0 {
		return fmt.Errorf("reset database rows but failed to remove some document files: %s", strings.Join(removeErrors, "; "))
	}

	return printJSON(summary)
}

func removeDocumentFiles(documentRoot string, relativePaths []string) ([]string, []string) {
	root := filepath.Clean(documentRoot)
	var removed []string
	var errors []string

	for _, relativePath := range relativePaths {
		relativePath = strings.TrimSpace(strings.ReplaceAll(relativePath, "\\", "/"))
		if relativePath == "" {
			continue
		}
		fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
		fullPath = filepath.Clean(fullPath)
		if !strings.HasPrefix(fullPath, root+string(os.PathSeparator)) && fullPath != root {
			errors = append(errors, fmt.Sprintf("refusing unsafe document path %q", relativePath))
			continue
		}
		if err := os.Remove(fullPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errors = append(errors, fmt.Sprintf("%s: %v", relativePath, err))
			continue
		}
		removed = append(removed, relativePath)
	}

	return removed, errors
}
