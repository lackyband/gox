package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lackyband/gox/domegaTrader"
)

func main() {
	// Parse command line arguments
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to configuration file")
	flag.Parse()

	if configFile == "" {
		log.Fatal("Please provide a configuration file using -config flag")
	}

	// Convert to absolute path
	absConfigFile, err := filepath.Abs(configFile)
	if err != nil {
		log.Fatalf("Failed to get absolute path for config file: %v", err)
	}

	// Check if config file exists
	if _, err := os.Stat(absConfigFile); os.IsNotExist(err) {
		log.Fatalf("Configuration file does not exist: %s", absConfigFile)
	}

	log.Printf("Starting domegaTrader with config: %s", absConfigFile)

	// Create configuration
	config := domegaTrader.NewConfig()
	config.ConfigFile = absConfigFile

	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create trading bot
	bot := domegaTrader.NewTradingBot(config)

	// Start the bot
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start trading bot: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	log.Println("Trading bot is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutdown signal received. Stopping trading bot...")

	// Stop the bot
	if err := bot.Stop(); err != nil {
		log.Printf("Error stopping trading bot: %v", err)
	}

	// Give some time for graceful shutdown
	time.Sleep(2 * time.Second)

	log.Println("domegaTrader stopped.")
}
