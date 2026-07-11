package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/seed"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func runSeedCommand(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("leotime seed", flag.ContinueOnError)
	force := flags.Bool("force", false, "fail if data already exists instead of skipping")
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
		return fmt.Errorf("load seed user %q: %w", email, err)
	}

	summary, err := seed.New(st).Run(ctx, seed.Options{
		UserID: user.ID,
		Force:  *force,
	})
	if err != nil {
		return err
	}
	return printJSON(summary)
}
