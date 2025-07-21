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
	// GetTorrents returns all torrents
	GetTorrentsCtx(ctx context.Context, o qbittorrent.TorrentFilterOptions) ([]qbittorrent.Torrent, error)

	// GetTorrentTrackers returns trackers for a specific torrent
	GetTorrentTrackersCtx(ctx context.Context, hash string) ([]qbittorrent.TorrentTracker, error)

	// ReannounceCtx reannounces a torrent to its trackers
	ReannounceTorrentWithRetry(ctx context.Context, hash string, o *qbittorrent.ReannounceOptions) (bool, error)

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
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

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

func (c *clientImpl) ReannounceTorrentWithRetry(ctx context.Context, hash string, o *qbittorrent.ReannounceOptions) (bool, error) {
	if ok, err := c.WaitForTrackerUpdate(ctx, hash); err != nil {
		return false, fmt.Errorf("failed to wait for tracker update: %w", err)
	} else if ok {
		return false, nil // No need to reannounce
	} else {
		if err := c.Client.ReannounceTorrentWithRetry(ctx, hash, o); err != nil {
			return false, fmt.Errorf("failed to reannounce torrent: %w", err)
		} else {
			return true, err // Successfully reannounced
		}
	}
}
