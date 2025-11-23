package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"teamacedia/discord-bot/internal/config"
	"teamacedia/discord-bot/internal/db"
	"teamacedia/discord-bot/internal/discord"
	"teamacedia/discord-bot/internal/sticky_roles"
)

func main() {
	// Load config file
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config.Config = cfg

	// Init DB
	err = sticky_roles.InitDB("sticky_roles.db")
	if err != nil {
		log.Fatal("Failed to init sticky_roles DB:", err)
	}
	err = db.InitDB("teamacedia.db")
	if err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}
	go discord.StartScheduler()

	// Start the Discord bot

	go discord.Start(cfg.Token, cfg.AppID, cfg.GuildID)

	// Channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until signal is received
	<-stop
	log.Println("Shutting down...")
}
