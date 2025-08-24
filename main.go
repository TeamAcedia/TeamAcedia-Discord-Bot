package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"teamacedia/discord-bot/internal/config"
	"teamacedia/discord-bot/internal/discord"
)

func main() {
	// Load config file
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config.Config = cfg

	// Start the Discord bot

	go discord.Start(cfg.Token, cfg.AppID, cfg.GuildID)

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until signal is received
	<-stop
	log.Println("Shutting down...")
}
