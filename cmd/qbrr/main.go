package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/autobrr/go-qbittorrent"

	"github.com/buroa/qbrr/pkg/client"
	"github.com/buroa/qbrr/pkg/config"
	"github.com/buroa/qbrr/pkg/logger"
	"github.com/buroa/qbrr/pkg/utils"
)

func process(ctx context.Context, client client.Client, torrent qbittorrent.Torrent, opts *config.Options) {
	if torrent.TimeActive > opts.MaxAge {
		slog.Debug("Torrent too old - skipping", "hash", torrent.Hash, "age", torrent.TimeActive)
		return
	}

	tracker, _ := utils.GetTLDPlusOne(torrent.Tracker)

	if utils.IsTrackerStatusOK(torrent.Trackers) {
		slog.Debug("Tracker OK - skipping", "hash", torrent.Hash, "tracker", tracker)
		return
	}

	if utils.IsTrackerStatusUpdating(torrent.Trackers) {
		slog.Debug("Waiting for tracker update", "hash", torrent.Hash, "tracker", tracker)

		timeout := time.Duration(opts.ReannounceOptions.Interval) * time.Second
		updateCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if ok, err := client.WaitForTrackerUpdateCtx(updateCtx, torrent.Hash); err != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				slog.Debug("Tracker update timed out - reannouncing", "hash", torrent.Hash, "tracker", tracker, "timeout", timeout)
			case errors.Is(err, context.Canceled):
				return // Exit if the context was canceled
			default:
				slog.Warn("Tracker update failed - reannouncing", "hash", torrent.Hash, "tracker", tracker, "error", err)
			}
		} else if ok {
			slog.Debug("Tracker update successful - skipping", "hash", torrent.Hash, "tracker", tracker)
			return
		}
	}

	if err := client.ReannounceTorrentWithRetry(ctx, torrent.Hash, &opts.ReannounceOptions); err != nil {
		slog.Warn("Reannounce failed", "hash", torrent.Hash, "tracker", tracker, "error", err)
	} else {
		slog.Info("Reannounced successfully", "hash", torrent.Hash, "tracker", tracker)
	}
}

func runAnnounce(ctx context.Context, client client.Client, opts *config.Options) error {
	torrents, err := client.GetTorrentsCtx(ctx, opts.TorrentFilterOptions)
	if err != nil {
		return fmt.Errorf("failed to get torrents: %w", err)
	}

	if len(torrents) == 0 && len(opts.TorrentFilterOptions.Hashes) > 0 {
		return fmt.Errorf("no torrent found for hash: %s", opts.TorrentFilterOptions.Hashes[0])
	}

	var wg sync.WaitGroup

	for _, torrent := range torrents {
		wg.Add(1)
		go func() {
			defer wg.Done()
			process(ctx, client, torrent, opts)
		}()
	}

	wg.Wait()

	return nil
}

func runDaemon(ctx context.Context, client client.Client, opts *config.Options) error {
	slog.Info("Starting torrent reannouncement daemon")

	ticker := time.NewTicker(time.Duration(opts.ReannounceOptions.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping torrent reannouncement daemon")
			return nil
		case <-ticker.C:
			if err := runAnnounce(ctx, client, opts); err != nil {
				slog.Error("Error during reannounce cycle", "error", err)
			}
		}
	}
}

func execute(ctx context.Context, client client.Client, opts *config.Options) error {
	if len(opts.TorrentFilterOptions.Hashes) > 0 {
		return runAnnounce(ctx, client, opts)
	}

	return runDaemon(ctx, client, opts)
}

func parseFlags() *config.Config {
	config := &config.Config{}

	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Int64Var(&config.MaxAge, "max-age", 300, "Maximum age of a torrent in seconds to reannounce")
	flag.IntVar(&config.MaxAttempts, "max-attempts", qbittorrent.ReannounceMaxAttempts, "Maximum number of reannounce attempts per torrent")
	flag.IntVar(&config.Interval, "interval", qbittorrent.ReannounceInterval, "Interval between reannounce checks in seconds")
	flag.StringVar(&config.Hash, "hash", "", "Specific torrent hash to reannounce (single run mode)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nqBittorrent reannouncement tool\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	return config
}

func main() {
	config := parseFlags()

	// Initialize logger
	logger.Initialize(config.LogLevel)

	// Create a context for the operations
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Listen for shutdown signals
	go func() {
		sig := <-sigs
		slog.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Create a new qBittorrent client
	client, err := client.NewClient()
	if err != nil {
		slog.Error("Failed to create qBittorrent client", "error", err)
		os.Exit(1)
	}

	// Create options from the config
	opts := config.ToOptions()

	// Execute the reannounce or daemon mode based on the provided options
	if err := execute(ctx, client, opts); err != nil {
		slog.Error("Failed to execute command", "error", err)
		os.Exit(1)
	}
}
