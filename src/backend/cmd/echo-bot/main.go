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

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/config"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/httpserver"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/maxbot"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/objectstorage"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/infrastructure/tarantool"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/control"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		return
	}

	bot, err := maxbot.New(cfg.MaxBotToken)
	if err != nil {
		logger.Error("create MAX client", "error", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	storageCtx, storageCancel := context.WithTimeout(ctx, 15*time.Second)
	storage, storageErr := tarantool.Connect(storageCtx, cfg.TarantoolAddress, cfg.TarantoolUser, cfg.TarantoolPassword)
	storageCancel()
	if storageErr != nil {
		logger.Warn("Tarantool unavailable, using in-memory repository until reconnect", "error", storageErr)
	} else {
		defer func() {
			if closeErr := storage.Close(); closeErr != nil {
				logger.Warn("close Tarantool", "error", closeErr)
			}
		}()
		logger.Info("Tarantool connected", "address", cfg.TarantoolAddress)
	}

	healthServer := &http.Server{
		Addr:              ":8080",
		Handler:           httpserver.HandlerWithVersion(bot.WebhookHandler(cfg.MaxWebhookSecret), cfg.ApplicationVersion),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if serveErr := healthServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("health server", "error", serveErr)
		}
	}()

	webhookURL := cfg.PublicBaseURL + "/webhook"
	if err := bot.Subscribe(ctx, webhookURL, cfg.MaxWebhookSecret); err != nil {
		logger.Error("subscribe MAX webhook", "url", webhookURL, "error", err)
		return
	}

	logger.Info("control bot started", "app", cfg.ApplicationName, "transport", "webhook", "url", webhookURL)
	repository := control.NewMemoryRepository(int64(cfg.InitialAdminMaxUserID))
	if storage != nil {
		loadCtx, loadCancel := context.WithTimeout(ctx, 5*time.Second)
		if users, loadErr := storage.Users(loadCtx); loadErr == nil {
			for _, user := range users {
				repository.SaveUser(user)
			}
		} else {
			logger.Warn("load users from Tarantool", "error", loadErr)
		}
		if tasks, loadErr := storage.Tasks(loadCtx); loadErr == nil {
			for _, task := range tasks {
				repository.SaveTask(task)
			}
		} else {
			logger.Warn("load tasks from Tarantool", "error", loadErr)
		}
		if invites, loadErr := storage.Invites(loadCtx); loadErr == nil {
			for _, invite := range invites {
				repository.SaveInvite(invite)
			}
		} else {
			logger.Warn("load invites from Tarantool", "error", loadErr)
		}
		if objects, loadErr := storage.Objects(loadCtx); loadErr == nil {
			for _, object := range objects {
				repository.SaveObject(object)
			}
		} else {
			logger.Warn("load objects from Tarantool", "error", loadErr)
		}
		if workTypes, loadErr := storage.WorkTypes(loadCtx); loadErr == nil {
			for _, workType := range workTypes {
				repository.SaveWorkType(workType)
			}
		} else {
			logger.Warn("load work types from Tarantool", "error", loadErr)
		}
		loadCancel()
	}
	var photoStore control.PhotoStore
	objectStore, objectStoreErr := objectstorage.New(cfg.ObjectStorageEndpoint, cfg.ObjectStorageAccessKey, cfg.ObjectStorageSecretKey, cfg.ObjectStorageBucket)
	if objectStoreErr != nil {
		logger.Warn("object storage unavailable, photo uploads disabled", "error", objectStoreErr)
	} else {
		bucketCtx, bucketCancel := context.WithTimeout(ctx, 15*time.Second)
		bucketErr := objectStore.EnsureBucket(bucketCtx)
		bucketCancel()
		if bucketErr != nil {
			logger.Warn("object storage bucket is not ready, photo uploads disabled", "error", bucketErr)
		} else {
			photoStore = objectStore
			logger.Info("object storage connected", "endpoint", cfg.ObjectStorageEndpoint, "bucket", cfg.ObjectStorageBucket)
		}
	}
	if runErr := control.NewService(bot, repository, photoStore, logger, storage).Run(ctx); runErr != nil && !errors.Is(runErr, context.Canceled) {
		logger.Error("echo bot stopped with error", "error", runErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("health server shutdown", "error", err)
	}
}
