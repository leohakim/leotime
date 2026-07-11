package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata"

	"github.com/leotime/leotime/apps/api/internal/backup"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/httpapi"
	"github.com/leotime/leotime/apps/api/internal/mail"
	_ "github.com/leotime/leotime/apps/api/internal/metrics"
	"github.com/leotime/leotime/apps/api/internal/notify"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/scheduler"
	"github.com/leotime/leotime/apps/api/internal/solidtimeimport"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "import" {
		if err := runImportCommand(context.Background(), os.Args[2:]); err != nil {
			log.Fatalf("import failed: %v", err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "backup" {
		if err := runBackupCommand(context.Background(), os.Args[2:]); err != nil {
			log.Fatalf("backup failed: %v", err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := runSeedCommand(context.Background(), os.Args[2:]); err != nil {
			log.Fatalf("seed failed: %v", err)
		}
		return
	}

	migrateOnly := flag.Bool("migrate-only", false, "apply database migrations and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	ctx := context.Background()

	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(ctx, database); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if *migrateOnly {
		log.Println("migrations applied")
		return
	}

	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, cfg.BootstrapEmail, cfg.BootstrapPassword); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	runCtx, cancelBackground := context.WithCancel(ctx)
	defer cancelBackground()

	mailSender, err := mail.NewSender(cfg)
	if err != nil {
		log.Fatalf("mail sender: %v", err)
	}

	outboxStore := outbox.NewStore(database)
	passwordReset := notify.NewPasswordResetService(st, outboxStore, cfg)
	notifier := notify.NewStillRunningNotifier(st, outboxStore, cfg)
	backupNotifier := notify.NewBackupNotifier(st, outboxStore, cfg)
	processor := outbox.NewProcessor(outboxStore, mailSender, outbox.ProcessorOptions{
		RetryPolicy: outbox.DefaultRetryPolicy(cfg.MailRetryBase, cfg.MailRetryMax),
		OnSent:      notifier.HandleSent,
	})
	backupService := backup.NewService(cfg, st, database, backupNotifier)
	backgroundScheduler := scheduler.New(cfg, st, notifier, processor, backupService)

	go func() {
		if cfg.SchedulerEnabled {
			log.Printf("scheduler enabled: scan=%s outbox=%s mail=%s", cfg.SchedulerScanInterval, cfg.OutboxProcessInterval, cfg.MailMode)
		} else {
			log.Printf("scheduler scan disabled; outbox=%s mail=%s", cfg.OutboxProcessInterval, cfg.MailMode)
		}
		if cfg.BackupSchedulerEnabled {
			log.Printf("backup scheduler enabled: interval=%s", cfg.BackupSchedulerInterval)
		}
		backgroundScheduler.Run(runCtx)
	}()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(cfg, st, passwordReset, backupService),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("leotime listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	cancelBackground()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func runImportCommand(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "solidtime" {
		return fmt.Errorf("usage: leotime import solidtime --file <zip> --user-email <email> [--dry-run]")
	}

	flags := flag.NewFlagSet("leotime import solidtime", flag.ContinueOnError)
	filePath := flags.String("file", "", "Solidtime ZIP export path")
	userEmail := flags.String("user-email", "", "leotime user email that will own imported records")
	dryRun := flags.Bool("dry-run", false, "validate and summarize without writing")
	if err := flags.Parse(args[1:]); err != nil {
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

	importer := solidtimeimport.New(database)
	summary, err := importer.ImportFile(ctx, solidtimeimport.Options{
		FilePath:  *filePath,
		UserEmail: *userEmail,
		DryRun:    *dryRun,
	})
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		return fmt.Errorf("print import summary: %w", err)
	}
	return nil
}
