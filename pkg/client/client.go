package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbr/pkg/utils"
)

// Client interface defines the contract for qBittorrent client operations
type Client interface {
	// Login authenticates with the qBittorrent server
	Login() error

	// GetTorrents returns all torrents
	GetTorrentsCtx(ctx context.Context, o qbittorrent.TorrentFilterOptions) ([]qbittorrent.Torrent, error)

	// GetTorrentTrackers returns trackers for a specific torrent
	GetTorrentTrackersCtx(ctx context.Context, hash string) ([]qbittorrent.TorrentTracker, error)

	// ReannounceCtx reannounces a torrent to its trackers
	ReannounceTorrentWithRetry(ctx context.Context, hash string, o *qbittorrent.ReannounceOptions) error

	// WaitForTrackerUpdate waits for tracker status to update
	WaitForTrackerUpdate(ctx context.Context, hash string) (bool, error)
}

// clientImpl wraps the qbittorrent.Client to implement our interface
type clientImpl struct {
	*qbittorrent.Client
}

var (
	defaultqBittorrentHost = "http://localhost:8080"
)

func NewClient() (Client, error) {
	host := os.Getenv("QBITTORRENT_HOST")
	if host == "" {
		if port := os.Getenv("QBT_WEBUI_PORT"); port != "" {
			host = fmt.Sprintf("http://localhost:%s", port)
		} else {
			host = defaultqBittorrentHost
		}
	}

	config := qbittorrent.Config{
		Host:     host,
		Username: os.Getenv("QBITTORRENT_USERNAME"),
		Password: os.Getenv("QBITTORRENT_PASSWORD"),
	}

	client := qbittorrent.NewClient(config)
	if err := client.Login(); err != nil {
		return nil, fmt.Errorf("failed to authenticate with qBittorrent: %w", err)
	}

	return &clientImpl{Client: client}, nil
}

func (c *clientImpl) WaitForTrackerUpdate(ctx context.Context, hash string) (bool, error) {
	if trackers, err := c.GetTorrentTrackersCtx(ctx, hash); err != nil {
		return false, fmt.Errorf("failed to get torrent trackers: %w", err)
	} else if len(trackers) == 0 {
		return false, fmt.Errorf("no trackers found for torrent: %s", hash)
	} else if !utils.IsTrackerStatusUpdating(trackers) {
		return utils.IsTrackerStatusOK(trackers), nil
	}

	// 1 second ticker to check tracker status
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// 60 second timeout
	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timeout:
			return false, fmt.Errorf("timeout waiting for tracker update to complete")
		case <-ticker.C:
			if trackers, err := c.GetTorrentTrackers(hash); err != nil {
				return false, fmt.Errorf("failed to get torrent trackers: %w", err)
			} else if !utils.IsTrackerStatusUpdating(trackers) {
				return utils.IsTrackerStatusOK(trackers), nil
			}
		}
	}
}
