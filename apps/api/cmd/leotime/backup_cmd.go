package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/leotime/leotime/apps/api/internal/backup"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func runBackupCommand(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: leotime backup run [--force] | list | restore --object-key <key> | restore --latest [--force]")
	}

	cfg := config.Load()
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

	user, err := st.UserByEmail(ctx, cfg.BootstrapEmail)
	if err != nil {
		return fmt.Errorf("load bootstrap user: %w", err)
	}

	service := backup.NewService(cfg, st, database)

	switch args[0] {
	case "run":
		flags := flag.NewFlagSet("leotime backup run", flag.ContinueOnError)
		force := flags.Bool("force", false, "run even if backup already succeeded today")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		result, err := service.Run(ctx, user.ID, *force)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "list":
		objects, err := service.ListObjects(ctx, user.ID)
		if err != nil {
			return err
		}
		return printJSON(map[string]any{"objects": objects})
	case "restore":
		flags := flag.NewFlagSet("leotime backup restore", flag.ContinueOnError)
		objectKey := flags.String("object-key", "", "S3 object key to restore")
		latest := flags.Bool("latest", false, "restore newest backup")
		force := flags.Bool("force", false, "skip confirmation")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if !*force {
			return fmt.Errorf("restore requires --force confirmation")
		}
		if *objectKey == "" && !*latest {
			return fmt.Errorf("restore requires --object-key or --latest")
		}
		result, err := service.Restore(ctx, user.ID, *objectKey, *latest)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return fmt.Errorf("unknown backup subcommand %q", args[0])
	}
}

func printJSON(payload any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("print json: %w", err)
	}
	return nil
}
