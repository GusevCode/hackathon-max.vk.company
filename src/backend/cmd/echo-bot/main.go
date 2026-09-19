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

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/httpserver"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/maxbot"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/echo"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		logger.Error("MAX_BOT_TOKEN is required")
		return
	}

	bot, err := maxbot.New(token)
	if err != nil {
		logger.Error("create MAX client", "error", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthServer := &http.Server{
		Addr:              ":8080",
		Handler:           httpserver.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if serveErr := healthServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("health server", "error", serveErr)
		}
	}()

	logger.Info("echo bot started", "transport", "long_polling")
	if runErr := echo.NewService(bot, logger).Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
		logger.Error("echo bot stopped with error", "error", runErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown", "error", err)
	}
}
