package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mbenaiss/whatsapp-mcp/api"
	"github.com/mbenaiss/whatsapp-mcp/config"
	"github.com/mbenaiss/whatsapp-mcp/db"
	"github.com/mbenaiss/whatsapp-mcp/services"
	"github.com/mbenaiss/whatsapp-mcp/whatsapp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	if err := os.MkdirAll(cfg.StoreDir, 0755); err != nil {
		log.Fatalf("Failed to create store directory: %v", err)
	}

	messageStore, err := db.NewDB(ctx, cfg.StoreDir)
	if err != nil {
		log.Fatalf("Failed to initialize message store: %v", err)
	}
	defer messageStore.Close()

	whatsappClient, err := whatsapp.NewWhatsapp(cfg.StoreDir)
	if err != nil {
		log.Fatalf("Failed to initialize WhatsApp client: %v", err)
	}

	service := services.NewService(whatsappClient, messageStore)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	apiServer := api.NewServer(service, cfg.Port)

	go func() {
		<-c
		log.Println("shutting down...")

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := apiServer.Stop(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}

		whatsappClient.Disconnect()
		log.Println("Server gracefully stopped")
	}()

	log.Printf("WhatsApp API server starting on port %s", cfg.Port)
	if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}
