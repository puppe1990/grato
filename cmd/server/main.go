package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/boot"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/grato/internal/app"
	appi18n "github.com/puppe1990/grato/internal/i18n"
	"github.com/puppe1990/grato/internal/store"
	"github.com/puppe1990/grato/web"
)

func main() {
	// Errors bubble out of run() instead of calling log.Fatal inline: a
	// log.Fatal after `defer stop()` skips the deferred signal cleanup
	// (gocritic exitAfterDefer).
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := cais.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}
	preferredPort := cfg.Port
	port, shifted, err := cais.ResolvePort(cfg.Port, cfg.Env)
	if err != nil {
		return err
	}
	cfg.Port = port

	a, err := bootstrapWithConfig(cfg)
	if err != nil {
		return err
	}

	shiftedFrom := ""
	if shifted {
		shiftedFrom = preferredPort
	}
	boot.Print(os.Stdout, boot.Options{
		AppName:         "grato",
		Config:          cfg,
		Version:         boot.CaisVersion(),
		PortShiftedFrom: shiftedFrom,
	})
	// Graceful shutdown on SIGINT/SIGTERM (air sends SIGINT before each
	// rebuild when send_interrupt is set) so the listener and sqlite close
	// instead of lingering as a zombie holding :8080 and data/app.db (#77).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.RunContext(ctx)
}

func bootstrapWithConfig(cfg cais.Config) (*app.App, error) {
	tmplFS, err := fs.Sub(web.Templates, "templates")
	if err != nil {
		return nil, fmt.Errorf("templates: %w", err)
	}

	catalog := appi18n.NewCatalog(cfg.Locale)
	views, err := view.Load(tmplFS, catalog)
	if err != nil {
		return nil, fmt.Errorf("views: %w", err)
	}

	s, err := store.NewSQLiteStore(cfg.DBPath, cfg.Env)
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}

	staticDir, err := cais.ResolveWebDir("static", cfg.StaticDir)
	if err != nil {
		_ = s.Close()
		return nil, err
	}

	return app.New(cfg, app.Deps{
		Views:     views,
		Store:     s,
		StaticDir: staticDir,
		Site:      meta.SiteFrom("grato", cfg.AppURL),
		Catalog:   catalog,
	})
}
