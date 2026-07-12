package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/leotime/leotime/apps/api/internal/enrich"
)

func runEnricherCommand(args []string) error {
	flags := flag.NewFlagSet("leotime enricher", flag.ContinueOnError)
	addr := flags.String("addr", "127.0.0.1:9333", "listen address")
	if err := flags.Parse(args); err != nil {
		return err
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: enrich.NewServer(),
	}
	log.Printf("leotime enricher listening on http://%s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("enricher serve: %w", err)
	}
	return nil
}
