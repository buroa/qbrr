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

type reannounceOptions struct {
	maxAge         int
	maxRetries     int
	interval       int
	processSeeding bool
}

func main() {
	var (
		logLevel       = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		maxAge         = flag.Int("max-age", 3600, "Maximum age of a torrent in seconds to reannounce")
		maxRetries     = flag.Int("max-retries", qbittorrent.ReannounceMaxAttempts, "Maximum number of reannounce retries per torrent")
		interval       = flag.Int("interval", qbittorrent.ReannounceInterval, "Interval between reannouncement attempts in seconds")
		processSeeding = flag.Bool("process-seeding", false, "Include seeding torrents in the reannouncement process")
	)

	flag.Parse()

	// Initialize logger
	logger.Initialize(*logLevel)

	// Create options struct
	opts := &reannounceOptions{
		maxAge:         *maxAge,
		maxRetries:     *maxRetries,
		interval:       *interval,
		processSeeding: *processSeeding,
	}

	// Run the reannounce logic
	if err := runReannounce(context.Background(), opts); err != nil {
		slog.Error("Failed to execute reannounce", "error", err)
		os.Exit(1)
	}
}

func runReannounce(ctx context.Context, opts *reannounceOptions) error {
	slog.Info("Starting torrent reannouncement process")

	client := qbittorrent.NewClient(qbittorrent.Config{
		Host:     os.Getenv("QBITTORRENT_HOST"),
		Username: os.Getenv("QBITTORRENT_USERNAME"),
		Password: os.Getenv("QBITTORRENT_PASSWORD"),
	})

	if err := client.Login(); err != nil {
		return fmt.Errorf("failed to authenticate with qBittorrent: %w", err)
	}

	for {
		filter := qbittorrent.TorrentFilterStalledDownloading
		if opts.processSeeding {
			filter = qbittorrent.TorrentFilterStalled
		}

		torrents, err := client.GetTorrents(qbittorrent.TorrentFilterOptions{
			Filter:          filter,
			IncludeTrackers: true,
		})
		if err != nil {
			return fmt.Errorf("failed to retrieve torrents: %w", err)
		}

		var wg sync.WaitGroup

		for _, torrent := range torrents {
			if shouldReannounce(torrent, opts.maxAge) {
				wg.Add(1)
				go func(t qbittorrent.Torrent) {
					defer wg.Done()
					if err := reannounceWithRetry(ctx, client, t, opts.maxRetries, opts.interval); err != nil {
						slog.Error("Failed to reannounce torrent", "hash", t.Hash, "error", err)
					} else {
						slog.Info("Reannounced torrent", "hash", t.Hash)
					}
				}(torrent)
			}
		}

		wg.Wait()

		time.Sleep(5 * time.Second)
	}
}

func shouldReannounce(torrent qbittorrent.Torrent, maxAge int) bool {
	if torrent.TimeActive > int64(maxAge) {
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

// reannounceWithRetry performs the reannounce operation with retry logic
func reannounceWithRetry(ctx context.Context, client *qbittorrent.Client, torrent qbittorrent.Torrent, maxAttempts, interval int) error {
	opts := qbittorrent.ReannounceOptions{
		Interval:        interval,
		MaxAttempts:     maxAttempts,
		DeleteOnFailure: false,
	}

	if err := client.ReannounceTorrentWithRetry(ctx, torrent.Hash, &opts); err != nil {
		if errors.Is(err, qbittorrent.ErrReannounceTookTooLong) {
			return fmt.Errorf("reannouncement timeout for torrent %s", torrent.Hash)
		}
		return fmt.Errorf("reannouncement failed for torrent %s: %w", torrent.Hash, err)
	}

	return nil
}
