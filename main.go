package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbr/internal/logger"
	"github.com/buroa/qbr/internal/utils"
)

var (
	defaultqBittorrentHost   = "http://localhost:8080"
	defaultMaxAttemptsDaemon = 3
)

type Config struct {
	LogLevel    string
	MaxAge      int64
	MaxAttempts int
	Interval    int
	Hash        string
}

type Options struct {
	maxAge               int64
	torrentFilterOptions qbittorrent.TorrentFilterOptions
	reannounceOptions    qbittorrent.ReannounceOptions
}

func (c *Config) ToOptions() *Options {
	opts := &Options{
		maxAge: c.MaxAge,
		torrentFilterOptions: qbittorrent.TorrentFilterOptions{
			Filter:          qbittorrent.TorrentFilterStalled,
			IncludeTrackers: true,
		},
		reannounceOptions: qbittorrent.ReannounceOptions{
			Interval:        c.Interval,
			MaxAttempts:     c.MaxAttempts,
			DeleteOnFailure: false,
		},
	}

	if c.Hash != "" {
		opts.torrentFilterOptions.Filter = qbittorrent.TorrentFilterAll
		opts.torrentFilterOptions.Hashes = []string{c.Hash}
	} else if !utils.FlagPassed("max-attempts") {
		opts.reannounceOptions.MaxAttempts = defaultMaxAttemptsDaemon
	}

	return opts
}

func NewClient() (*qbittorrent.Client, error) {
	config := qbittorrent.Config{
		Host:     os.Getenv("QBITTORRENT_HOST"),
		Username: os.Getenv("QBITTORRENT_USERNAME"),
		Password: os.Getenv("QBITTORRENT_PASSWORD"),
	}

	if config.Host == "" {
		if port := os.Getenv("QBT_WEBUI_PORT"); port != "" {
			config.Host = fmt.Sprintf("http://localhost:%s", port)
		} else {
			config.Host = defaultqBittorrentHost
		}
	}

	client := qbittorrent.NewClient(config)
	if err := client.Login(); err != nil {
		return nil, fmt.Errorf("failed to authenticate with qBittorrent: %w", err)
	}

	return client, nil
}

func runReannounce(ctx context.Context, client *qbittorrent.Client, opts *Options) error {
	torrents, err := client.GetTorrentsCtx(ctx, opts.torrentFilterOptions)
	if err != nil {
		return fmt.Errorf("failed to retrieve torrents: %w", err)
	}

	if len(opts.torrentFilterOptions.Hashes) != 0 && len(torrents) == 0 {
		return fmt.Errorf("no torrent found with hash %s", opts.torrentFilterOptions.Hashes[0])
	}

	var wg sync.WaitGroup

	for _, torrent := range torrents {
		if utils.ShouldReannounce(torrent, opts.maxAge) {
			wg.Add(1)
			go func(t qbittorrent.Torrent) {
				defer wg.Done()
				if err := client.ReannounceTorrentWithRetry(ctx, t.Hash, &opts.reannounceOptions); err != nil {
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

	ticker := time.NewTicker(time.Duration(opts.reannounceOptions.Interval) * time.Second)
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

func execute(ctx context.Context, client *qbittorrent.Client, opts *Options) error {
	if len(opts.torrentFilterOptions.Hashes) != 0 {
		return runReannounce(ctx, client, opts)
	}

	return runDaemon(ctx, client, opts)
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Int64Var(&config.MaxAge, "max-age", 120, "Maximum age of a torrent in seconds to reannounce")
	flag.IntVar(&config.MaxAttempts, "max-attempts", qbittorrent.ReannounceMaxAttempts, "Maximum number of reannounce attempts per torrent")
	flag.IntVar(&config.Interval, "interval", qbittorrent.ReannounceInterval, "Interval between reannounce checks in seconds (daemon mode)")
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

	// Create a new qBittorrent client
	client, err := NewClient()
	if err != nil {
		slog.Error("Failed to create qBittorrent client", "error", err)
		os.Exit(1)
	}

	// Create options from the config
	opts := config.ToOptions()

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

	// Execute the reannounce or daemon mode based on the provided options
	if err := execute(ctx, client, opts); err != nil {
		slog.Error("Failed to execute command", "error", err)
		os.Exit(1)
	}
}
