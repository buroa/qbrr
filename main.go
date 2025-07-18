package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbr/internal/logger"
)

type Options struct {
	maxAge     int64
	maxRetries int
	interval   int
}

func main() {
	var (
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		maxAge     = flag.Int64("max-age", 900, "Maximum age of a torrent in seconds to reannounce")
		maxRetries = flag.Int("max-retries", 5, "Maximum number of reannounce retries per torrent")
		interval   = flag.Int("interval", qbittorrent.ReannounceInterval, "Interval between reannouncement attempts in seconds")
	)

	flag.Parse()

	// Initialize logger
	logger.Initialize(*logLevel)

	// Create options struct
	opts := &Options{
		maxAge:     *maxAge,
		maxRetries: *maxRetries,
		interval:   *interval,
	}

	// Run the reannounce logic
	if err := runReannounce(context.Background(), opts); err != nil {
		slog.Error("Failed to execute reannounce", "error", err)
		os.Exit(1)
	}
}

func runReannounce(ctx context.Context, opts *Options) error {
	slog.Info("Starting torrent reannouncement process")

	client := qbittorrent.NewClient(qbittorrent.Config{
		Host:     os.Getenv("QBITTORRENT_HOST"),
		Username: os.Getenv("QBITTORRENT_USERNAME"),
		Password: os.Getenv("QBITTORRENT_PASSWORD"),
	})

	if err := client.Login(); err != nil {
		return fmt.Errorf("failed to authenticate with qBittorrent: %w", err)
	}

	torrentFilterOptions := qbittorrent.TorrentFilterOptions{
		Filter:          qbittorrent.TorrentFilterStalled,
		IncludeTrackers: true,
	}

	reannounceOptions := qbittorrent.ReannounceOptions{
		Interval:        opts.interval,
		MaxAttempts:     opts.maxRetries,
		DeleteOnFailure: false,
	}

	for {
		torrents, err := client.GetTorrents(torrentFilterOptions)

		if err != nil {
			slog.Error("Failed to retrieve torrents", "error", err)
			continue
		}

		var reannounceCount int
		var wg sync.WaitGroup

		for _, torrent := range torrents {
			if shouldReannounce(torrent, opts.maxAge) {
				reannounceCount++
				wg.Add(1)
				go func(t qbittorrent.Torrent) {
					defer wg.Done()
					if err := reannounceWithRetry(ctx, client, t, &reannounceOptions); err != nil {
						slog.Warn("Failed to reannounce torrent", "name", t.Name, "hash", t.Hash, "error", err)
					} else {
						slog.Info("Reannounced torrent", "name", t.Name, "hash", t.Hash)
					}
				}(torrent)
			}
		}

		wg.Wait()

		if reannounceCount == 0 {
			time.Sleep(5 * time.Second)
		}
	}
}

func shouldReannounce(torrent qbittorrent.Torrent, maxAge int64) bool {
	if torrent.TimeActive > maxAge {
		return false
	}

	if torrent.NumSeeds > 0 || torrent.NumLeechs > 0 {
		return false
	}

	for _, tracker := range torrent.Trackers {
		if tracker.Status == qbittorrent.TrackerStatusOK {
			return false
		}
	}

	return true
}

func reannounceWithRetry(ctx context.Context, client *qbittorrent.Client, torrent qbittorrent.Torrent, opts *qbittorrent.ReannounceOptions) error {
	if err := client.ReannounceTorrentWithRetry(ctx, torrent.Hash, opts); err != nil {
		if errors.Is(err, qbittorrent.ErrReannounceTookTooLong) {
			return fmt.Errorf("reannouncement timeout for torrent %s", torrent.Hash)
		}
		return fmt.Errorf("reannouncement failed for torrent %s: %w", torrent.Hash, err)
	}

	return nil
}
