package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"personal-grammarly/action"
	"personal-grammarly/ai"
	"personal-grammarly/config"
	"personal-grammarly/hotkey"
)

func main() {
	log.Println("Starting Personal Grammarly...")

	cfg := config.Load()

	if cfg.APIKey == "" {
		log.Println("WARNING: API_KEY is empty. The application will not be able to correct text until you set it in the .env file.")
	}

	aiClient := ai.NewClient(cfg.APIKey, cfg.APIBaseURL, cfg.ModelName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Start hotkey listener
	err := hotkey.Listen(ctx, cfg.HotkeyMod, cfg.HotkeyKey, func() {
		// Run action asynchronously so we don't block the hotkey listener
		go action.RunCorrection(aiClient)
	})

	if err != nil {
		log.Fatalf("Failed to start hotkey listener: %v", err)
	}
}
