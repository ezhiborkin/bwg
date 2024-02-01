package main

import (
	"billing/internal/config"
	"billing/internal/http-server/handlers"
	"billing/internal/readers/invoiceReader"
	"billing/internal/readers/withdrawReader"
	"billing/internal/service"
	"billing/internal/storage/postgresql"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info(
		"starting url-shortener",
		slog.String("env", cfg.Env),
		slog.String("version", "123"),
	)
	log.Debug("debug messages are enabled")

	repo, err := postgresql.New(cfg.DataSourceName)
	if err != nil {
		log.Error("failed to initialize storage", err)
		os.Exit(1)
	}

	service := service.New(log, repo, repo, repo, repo)

	withdrawReader := withdrawReader.New(service)
	invoiceReader := invoiceReader.New(service)

	go withdrawReader.Read()
	go invoiceReader.Read()

	handler := handlers.New(service, service, service)

	router := handler.InitRoutes()

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:        cfg.HTTPServer.Address,
		Handler:     router,
		ReadTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout: cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("failed to start server")
		}
	}()

	log.Info("server started")

	<-done
	log.Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPServer.Timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to stop server", err)

		return
	}

	log.Info("server stopped")

	// err = repo.CreateWallet()
	// if err != nil {
	// 	log.Error("failed to create wallet", err)
	// 	os.Exit(1)
	// }

	// var balances []balance.BalanceResponse
	// balances, err = repo.GetBalance("")

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default: // If env config is invalid, set prod settings by default due to security
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
