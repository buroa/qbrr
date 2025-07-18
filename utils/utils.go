package utils

import (
	"strings"

	"github.com/autobrr/go-qbittorrent"
)

var (
	words = []string{"unregistered", "not registered", "not found", "not exist"}
)

func ShouldReannounce(torrent qbittorrent.Torrent, maxAge int64) bool {
	if torrent.TimeActive > maxAge {
		return false
	}

	if torrent.NumSeeds > 0 || torrent.NumLeechs > 0 {
		return false
	}

	return !isTrackerStatusOK(torrent.Trackers)
}

func isTrackerStatusOK(trackers []qbittorrent.TorrentTracker) bool {
	for _, tracker := range trackers {
		if tracker.Status == qbittorrent.TrackerStatusDisabled {
			continue
		}

		// check for certain messages before the tracker status to catch ok status with unreg msg
		if isUnregistered(tracker.Message) {
			return false
		}

		if tracker.Status == qbittorrent.TrackerStatusOK {
			return true
		}
	}

	return false
}

func isUnregistered(msg string) bool {
	msg = strings.ToLower(msg)

	for _, v := range words {
		if strings.Contains(msg, v) {
			return true
		}
	}

	return false
}
