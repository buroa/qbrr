package utils

import (
	"strings"

	"github.com/autobrr/go-qbittorrent"
)

var (
	words = []string{"unregistered", "not registered", "not found", "not exist"}
)

func IsTrackerStatusOK(trackers []qbittorrent.TorrentTracker) bool {
	for _, tracker := range trackers {
		switch tracker.Status {
		case qbittorrent.TrackerStatusDisabled:
			continue
		case qbittorrent.TrackerStatusOK:
			if IsUnregistered(tracker.Message) {
				continue
			}
			return true
		default:
			continue
		}
	}

	return false
}

func IsTrackerStatusUpdating(trackers []qbittorrent.TorrentTracker) bool {
	for _, tracker := range trackers {
		switch tracker.Status {
		case qbittorrent.TrackerStatusDisabled:
			continue
		case qbittorrent.TrackerStatusUpdating, qbittorrent.TrackerStatusNotContacted:
			continue
		default:
			return false
		}
	}

	return true
}

func IsUnregistered(msg string) bool {
	msg = strings.ToLower(msg)

	for _, v := range words {
		if strings.Contains(msg, v) {
			return true
		}
	}

	return false
}
