package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kapombe-k/Biostat/internal/clients"
	"github.com/kapombe-k/Biostat/internal/config"
	"github.com/kapombe-k/Biostat/internal/handlers"
	"github.com/kapombe-k/Biostat/internal/services"
	"github.com/kapombe-k/Biostat/internal/templates"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid BioScope configuration", "err", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	renderer, err := templates.NewRenderer()
	if err != nil {
		logger.Error("failed to compile HTML templates", "err", err)
		os.Exit(1)
	}

	safeHTTP := clients.NewSafeHTTPClient(cfg, logger)
	uniprotClient := clients.NewUniProtClient(safeHTTP, cfg.UniProtBaseURL)
	rcsbClient := clients.NewRCSBClient(safeHTTP, cfg.RCSBSearchURL, cfg.RCSBDataURL, cfg.RCSBFilesURL)
	afClient := clients.NewAlphaFoldClient(safeHTTP, cfg.AlphaFoldBaseURL)
	pdbeClient := clients.NewPDBeClient(safeHTTP, cfg.PDBeBaseURL)
	blastClient := clients.NewBlastClient(safeHTTP, cfg)

	blastMgr := services.NewBlastManager(cfg, blastClient, logger)
	explorerSvc := services.NewExplorerService(cfg, uniprotClient, rcsbClient, afClient, pdbeClient, blastMgr, logger)
	srv := handlers.NewServer(cfg, explorerSvc, blastMgr, renderer, "static", logger)

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("BioScope server listening",
			"addr", cfg.Addr(),
			"offline_fallback", cfg.OfflineFallback,
			"cache_max_items", cfg.CacheMaxItems,
		)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		logger.Info("shutting down BioScope server gracefully", "signal", sig.String())
	case serveErr := <-errCh:
		logger.Error("HTTP server failed", "err", serveErr)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Warn("graceful shutdown warning", "err", err)
	}
}
