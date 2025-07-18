package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbr/internal/logger"
	"github.com/buroa/qbr/utils"
)

type Options struct {
	maxAge   int64
	interval int
}

func main() {
	var (
		logLevel = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		maxAge   = flag.Int64("max-age", 900, "Maximum age of a torrent in seconds to reannounce")
		interval = flag.Int("interval", 7, "Interval between reannounce checks in seconds")
	)

	flag.Parse()

	// Initialize logger
	logger.Initialize(*logLevel)

	// Create options struct
	opts := &Options{
		maxAge:   *maxAge,
		interval: *interval,
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

	for {
		torrents, err := client.GetTorrents(torrentFilterOptions)

		if err != nil {
			slog.Error("Failed to retrieve torrents", "error", err)
			continue
		}

		var reannounceHashes []string

		for _, torrent := range torrents {
			if utils.ShouldReannounce(torrent, opts.maxAge) {
				reannounceHashes = append(reannounceHashes, torrent.Hash)
			}
		}

		if len(reannounceHashes) > 0 {
			if err := client.ReAnnounceTorrentsCtx(ctx, reannounceHashes); err != nil {
				slog.Error("Failed to reannounce torrents", "error", err)
			} else {
				slog.Info("Reannounced torrents", "hashes", reannounceHashes)
			}
		}

		time.Sleep(time.Duration(opts.interval) * time.Second)
	}
}
