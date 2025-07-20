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

var (
	torrentFilterOptions = qbittorrent.TorrentFilterOptions{
		Filter:          qbittorrent.TorrentFilterStalled,
		IncludeTrackers: true,
	}

	qbittorrentConfig = qbittorrent.Config{
		Host:     os.Getenv("QBITTORRENT_HOST"),
		Username: os.Getenv("QBITTORRENT_USERNAME"),
		Password: os.Getenv("QBITTORRENT_PASSWORD"),
	}
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
		maxAge      = flag.Int64("max-age", 900, "Maximum age of a torrent in seconds to reannounce")
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

	// Validate options
	if opts.hash != "" && !utils.FlagPassed("max-attempts") {
		opts.maxAttempts = qbittorrent.ReannounceMaxAttempts
	}

	// Run the reannounce logic
	if err := runReannounce(context.Background(), opts); err != nil {
		slog.Error("Failed to execute reannounce", "error", err)
		os.Exit(1)
	}
}

func runReannounce(ctx context.Context, opts *Options) error {
	client := qbittorrent.NewClient(qbittorrentConfig)
	if err := client.Login(); err != nil {
		return fmt.Errorf("failed to authenticate with qBittorrent: %w", err)
	}

	reannounceOptions := qbittorrent.ReannounceOptions{
		Interval:        opts.interval,
		MaxAttempts:     opts.maxAttempts,
		DeleteOnFailure: false,
	}

	if opts.hash != "" {
		torrentFilterOptions.Filter = qbittorrent.TorrentFilterAll
		torrentFilterOptions.Hashes = []string{opts.hash}
	} else {
		slog.Info("Starting torrent reannouncement process")
	}

	for {
		torrents, err := client.GetTorrents(torrentFilterOptions)
		if err != nil {
			slog.Error("Failed to retrieve torrents", "error", err)
			continue
		}

		if opts.hash != "" && len(torrents) == 0 {
			return fmt.Errorf("no torrents found with hash %s", opts.hash)
		}

		var reannounceCount int
		var wg sync.WaitGroup

		for _, torrent := range torrents {
			if utils.ShouldReannounce(torrent, opts.maxAge) {
				reannounceCount++
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

		if opts.hash != "" {
			return nil
		}

		if reannounceCount == 0 {
			time.Sleep(time.Duration(opts.interval) * time.Second)
		}
	}
}
