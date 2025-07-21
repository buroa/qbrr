package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbr/pkg/utils"
)

type Client interface {
	Login() error
	GetTorrentsCtx(ctx context.Context, o qbittorrent.TorrentFilterOptions) ([]qbittorrent.Torrent, error)
	GetTorrentTrackersCtx(ctx context.Context, hash string) ([]qbittorrent.TorrentTracker, error)
	ReannounceTorrentWithRetry(ctx context.Context, hash string, o *qbittorrent.ReannounceOptions) error
	WaitForTrackerUpdate(ctx context.Context, hash string) (bool, error)
}

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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			if trackers, err := c.GetTorrentTrackers(hash); err != nil {
				return false, fmt.Errorf("failed to get torrent trackers: %w", err)
			} else if len(trackers) == 0 {
				return false, fmt.Errorf("no trackers found for hash: %s", hash)
			} else if !utils.IsTrackerStatusUpdating(trackers) {
				return utils.IsTrackerStatusOK(trackers), nil
			}
		}
	}
}
