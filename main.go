package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbr/internal/logger"
	"github.com/buroa/qbr/utils"
)

var (
	defaultTorrentFilterOptions = qbittorrent.TorrentFilterOptions{
		Filter:          qbittorrent.TorrentFilterStalled,
		IncludeTrackers: true,
	}
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

	for {
		torrents, err := client.GetTorrents(defaultTorrentFilterOptions)

		if err != nil {
			slog.Error("Failed to retrieve torrents", "error", err)
			continue
		}

		var hashes []string

		for _, torrent := range torrents {
			if utils.ShouldReannounce(torrent, opts.maxAge) {
				hashes = append(hashes, torrent.Hash)
			}
		}

		if len(hashes) > 0 {
			if err := client.ReAnnounceTorrentsCtx(ctx, hashes); err != nil {
				slog.Error("Failed to reannounce torrents", "error", err)
				continue
			}

			if len(hashes) == 1 {
				slog.Info("Reannounced torrent", "hash", hashes[0])
			} else {
				slog.Info("Reannounced torrents", "hashes", strings.Join(hashes, ", "))
			}
		}

		time.Sleep(time.Duration(opts.interval) * time.Second)
	}
}
