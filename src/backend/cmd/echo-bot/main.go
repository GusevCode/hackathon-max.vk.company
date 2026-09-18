package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/health"
	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func main() {
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		log.Fatal("MAX_BOT_TOKEN is required")
	}

	api, err := maxbot.New(token)
	if err != nil {
		log.Fatalf("create MAX client: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthServer := &http.Server{
		Addr:              ":8080",
		Handler:           health.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := healthServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("health server: %v", err)
		}
	}()

	log.Println("echo bot started (long polling)")
	for update := range api.GetUpdates(ctx) {
		switch update := update.(type) {
		case *schemes.MessageCreatedUpdate:
			if update.GetText() == "" {
				continue
			}

			err := api.Messages.Send(
				ctx,
				maxbot.NewMessage().
					SetChat(update.GetChatID()).
					SetText(update.GetText()),
			)
			if err != nil {
				log.Printf("send echo: %v", err)
			}
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("health server shutdown: %v", err)
	}

	log.Println("echo bot stopped")
}
