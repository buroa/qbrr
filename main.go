package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbr/internal/logger"
	"github.com/buroa/qbr/utils"
)

type Options struct {
	maxAge      int64
	maxAttempts int
	interval    int
	hash        string
}

func main() {
	var (
		logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		maxAge      = flag.Int64("max-age", 120, "Maximum age of a torrent in seconds to reannounce")
		maxAttempts = flag.Int("max-attempts", 3, "Maximum number of reannounce attempts per torrent")
		interval    = flag.Int("interval", 7, "Interval between reannounce checks in seconds")
		hash        = flag.String("hash", "", "Specific torrent hash to reannounce")
	)

	flag.Parse()

	// Initialize logger
	logger.Initialize(*logLevel)

	// Create options struct
	opts := &Options{
		maxAge:      *maxAge,
		maxAttempts: *maxAttempts,
		interval:    *interval,
		hash:        *hash,
	}

	// Set up qBittorrent client configuration
	qbittorrentConfig := qbittorrent.Config{
		Host:     os.Getenv("QBITTORRENT_HOST"),
		Username: os.Getenv("QBITTORRENT_USERNAME"),
		Password: os.Getenv("QBITTORRENT_PASSWORD"),
	}

	// Automatically set host if not provided
	if qbittorrentConfig.Host == "" {
		qbittorrentHost := "http://localhost"
		if qbittorrentPort := os.Getenv("QBT_WEBUI_PORT"); qbittorrentPort != "" {
			qbittorrentHost += ":" + qbittorrentPort
		}
		qbittorrentConfig.Host = qbittorrentHost
	}

	// Create qBittorrent client
	client := qbittorrent.NewClient(qbittorrentConfig)
	if err := client.Login(); err != nil {
		slog.Error("Failed to connect to qBittorrent", "error", err)
		os.Exit(1)
	}

	// Create a context for the operations
	ctx := context.Background()

	// Run the reannounce logic
	if opts.hash != "" {
		if err := runReannounce(ctx, client, opts); err != nil {
			slog.Error("Failed to reannounce torrent", "error", err)
			os.Exit(1)
		}
	} else {
		if err := runDaemon(ctx, client, opts); err != nil {
			slog.Error("Failed to start daemon", "error", err)
			os.Exit(1)
		}
	}
}

func runReannounce(ctx context.Context, client *qbittorrent.Client, opts *Options) error {
	torrentFilterOptions := qbittorrent.TorrentFilterOptions{
		Filter:          qbittorrent.TorrentFilterStalled,
		IncludeTrackers: true,
	}

	reannounceOptions := qbittorrent.ReannounceOptions{
		Interval:        opts.interval,
		MaxAttempts:     opts.maxAttempts,
		DeleteOnFailure: false,
	}

	if opts.hash != "" {
		torrentFilterOptions.Filter = qbittorrent.TorrentFilterAll
		torrentFilterOptions.Hashes = []string{opts.hash}

		if !utils.FlagPassed("max-attempts") {
			reannounceOptions.MaxAttempts = qbittorrent.ReannounceMaxAttempts
		}
	}

	torrents, err := client.GetTorrents(torrentFilterOptions)
	if err != nil {
		return fmt.Errorf("failed to retrieve torrents: %w", err)
	}

	if opts.hash != "" && len(torrents) == 0 {
		return fmt.Errorf("no torrent found with hash %s", opts.hash)
	}

	var wg sync.WaitGroup

	for _, torrent := range torrents {
		if utils.ShouldReannounce(torrent, opts.maxAge) {
			wg.Add(1)
			go func(t qbittorrent.Torrent) {
				defer wg.Done()
				if err := client.ReannounceTorrentWithRetry(ctx, t.Hash, &reannounceOptions); err != nil {
					slog.Error("Failed to reannounce torrent", "name", t.Name, "hash", t.Hash, "error", err)
				} else {
					slog.Info("Reannounced torrent", "name", t.Name, "hash", t.Hash)
				}
			}(torrent)
		}
	}

	wg.Wait()

	return nil
}

func runDaemon(ctx context.Context, client *qbittorrent.Client, opts *Options) error {
	slog.Info("Starting torrent reannouncement daemon")

	ticker := time.NewTicker(time.Duration(opts.interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping torrent reannouncement daemon")
			return nil
		case <-ticker.C:
			if err := runReannounce(ctx, client, opts); err != nil {
				slog.Error("Error during reannounce cycle", "error", err)
			}
		}
	}
}
