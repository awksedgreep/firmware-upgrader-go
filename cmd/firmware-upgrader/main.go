package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/awksedgreep/firmware-upgrader/internal/api"
	"github.com/awksedgreep/firmware-upgrader/internal/database"
	"github.com/awksedgreep/firmware-upgrader/internal/engine"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	version   = "0.1.0"
	buildTime = "unknown"
)

func main() {
	// Command line flags
	var (
		dbPath   = flag.String("db", "upgrader.db", "Path to SQLite database")
		bind     = flag.String("bind", "0.0.0.0", "Bind address/interface (e.g., 127.0.0.1, 192.168.1.1, or 0.0.0.0)")
		port     = flag.Int("port", 8080, "HTTP server port")
		logLevel = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		workers  = flag.Int("workers", 0, "Number of concurrent upgrade workers (0 = use database setting)")
		showVer  = flag.Bool("version", false, "Show version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("Firmware Upgrader v%s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Configure logging
	setupLogging(*logLevel)

	log.Info().
		Str("version", version).
		Str("db_path", *dbPath).
		Str("bind", *bind).
		Int("port", *port).
		Msg("Starting Firmware Upgrader")

	// Initialize database
	db, err := database.New(*dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	log.Info().Str("path", *dbPath).Msg("Database initialized successfully")

	// Load settings from database
	settings, err := db.ListSettings()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load settings from database")
	}

	// Parse settings with command-line overrides
	workersCount := *workers
	if workersCount == 0 {
		if w, err := strconv.Atoi(settings["workers"]); err == nil {
			workersCount = w
		} else {
			workersCount = 5
		}
	}

	discoveryInterval, _ := strconv.Atoi(settings["discovery_interval"])
	jobTimeout, _ := strconv.Atoi(settings["job_timeout"])
	retryAttempts, _ := strconv.Atoi(settings["retry_attempts"])
	maxPerCMTS, _ := strconv.Atoi(settings["max_upgrades_per_cmts"])

	if discoveryInterval == 0 {
		discoveryInterval = 60
	}
	if jobTimeout == 0 {
		jobTimeout = 300
	}
	if retryAttempts == 0 {
		retryAttempts = 3
	}
	if maxPerCMTS == 0 {
		maxPerCMTS = 10
	}

	log.Info().
		Int("workers", workersCount).
		Int("discovery_interval", discoveryInterval).
		Int("job_timeout", jobTimeout).
		Int("retry_attempts", retryAttempts).
		Int("max_per_cmts", maxPerCMTS).
		Msg("Settings loaded from database")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize upgrade engine
	eng := engine.New(db, engine.Config{
		Workers:       workersCount,
		RetryAttempts: retryAttempts,
		PollInterval:  time.Duration(discoveryInterval) * time.Second,
		JobTimeout:    time.Duration(jobTimeout) * time.Second,
		MaxPerCMTS:    maxPerCMTS,
	})

	// Start engine in background
	go func() {
		if err := eng.Start(ctx); err != nil {
			log.Error().Err(err).Msg("Upgrade engine error")
		}
	}()

	log.Info().Int("workers", workersCount).Msg("Upgrade engine started")

	// Initialize API server
	srv := api.NewServer(db, eng, api.Config{
		Bind:    *bind,
		Port:    *port,
		WebRoot: "./web",
	})

	// Start server in background
	go func() {
		log.Info().Str("bind", *bind).Int("port", *port).Msgf("HTTP server listening on http://%s:%d", *bind, *port)
		if err := srv.Start(); err != nil {
			log.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutdown signal received, gracefully shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Cancel engine context
	cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}

	log.Info().Msg("Firmware Upgrader shut down gracefully")
}

func setupLogging(level string) {
	// Pretty console logging for development
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})

	// Set log level
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Str("provided", level).Msg("Unknown log level, using 'info'")
	}
}
