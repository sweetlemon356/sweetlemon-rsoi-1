package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/app"
	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/config"
	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/httpapi"
	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("parse database configuration: %w", err)
	}
	poolConfig.MaxConns = cfg.Database.MaxConnections

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	connectCtx, cancelConnect := context.WithTimeout(ctx, cfg.Database.ConnectTimeout)
	err = pool.Ping(connectCtx)
	cancelConnect()
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	repository := postgres.NewPersonRepository(pool)
	service := app.NewService(repository)
	handler := httpapi.NewHandler(service)
	router := httpapi.NewRouter(handler, pool)

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	serverError := make(chan error, 1)
	go func() {
		slog.Info("server started",
			"address", cfg.HTTP.Address,
			"swagger", "/swagger/",
		)
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutting down server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}
