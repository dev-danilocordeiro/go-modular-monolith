package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-danilocordeiro/go-modular-monolith/internal/bootstrap"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/config"
	"github.com/dev-danilocordeiro/go-modular-monolith/internal/platform/database"
)

// version é preenchida no build: -ldflags "-X main.version=..."
var version = "dev"

func main() {
	// A imagem distroless não tem shell nem curl: o HEALTHCHECK do Docker
	// chama o próprio binário com este subcomando.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("version", version)
	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

// run concentra a lógica do main e devolve erro em vez de chamar os.Exit
// espalhado, assim os defers sempre rodam.
func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var db *pgxpool.Pool
	if cfg.DBDriver == "postgres" {
		db, err = database.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()
		if err := database.Migrate(ctx, db); err != nil {
			return err
		}
	}

	app := bootstrap.NewApp(log, db, cfg.ProblemTypeBase)

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.HTTPAddr, "db", cfg.DBDriver)
		errCh <- app.Listen(cfg.HTTPAddr, fiber.ListenConfig{DisableStartupMessage: true})
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return nil
	}
}

func healthcheck() error {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("healthcheck: HTTP_ADDR inválido: %w", err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort(host, port)+"/health", nil)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: status %d", resp.StatusCode)
	}
	return nil
}
