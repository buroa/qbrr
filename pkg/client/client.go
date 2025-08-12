package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/autobrr/go-qbittorrent"
	"github.com/buroa/qbrr/pkg/utils"
)

type Client interface {
	Login() error
	GetTorrentsCtx(ctx context.Context, o qbittorrent.TorrentFilterOptions) ([]qbittorrent.Torrent, error)
	GetTorrentTrackersCtx(ctx context.Context, hash string) ([]qbittorrent.TorrentTracker, error)
	ReannounceTorrentWithRetry(ctx context.Context, hash string, o *qbittorrent.ReannounceOptions) error
	WaitForTrackerUpdateCtx(ctx context.Context, hash string) (bool, error)
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

func (c *clientImpl) WaitForTrackerUpdateCtx(ctx context.Context, hash string) (bool, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			if trackers, err := c.GetTorrentTrackersCtx(ctx, hash); err != nil {
				// Check if error is due to context cancellation/timeout first,
				// as the underlying library may wrap the original context error
				if ctx.Err() != nil {
					return false, ctx.Err()
				}
				return false, err
			} else if len(trackers) == 0 {
				return false, fmt.Errorf("no trackers found for hash: %s", hash)
			} else if !utils.IsTrackerStatusUpdating(trackers) {
				return utils.IsTrackerStatusOK(trackers), nil
			}
		}
	}
}
