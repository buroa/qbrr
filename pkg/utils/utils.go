package utils

import (
	"net/url"
	"strings"

	"github.com/autobrr/go-qbittorrent"
	"golang.org/x/net/publicsuffix"
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

	return !IsTrackerStatusOK(torrent.Trackers)
}

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

func GetTLDPlusOne(tracker string) (string, error) {
	parsedURL, err := url.Parse(tracker)
	if err != nil {
		return tracker, err
	}

	tldPlusOne, err := publicsuffix.EffectiveTLDPlusOne(parsedURL.Hostname())
	if err != nil {
		return tracker, err
	}

	return strings.ToLower(tldPlusOne), nil
}
