package config

import (
	"github.com/autobrr/go-qbittorrent"
)

var (
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
	MaxAge               int64
	TorrentFilterOptions qbittorrent.TorrentFilterOptions
	ReannounceOptions    qbittorrent.ReannounceOptions
}

func (c *Config) ToOptions() *Options {
	opts := &Options{
		MaxAge: c.MaxAge,
		TorrentFilterOptions: qbittorrent.TorrentFilterOptions{
			Filter:          qbittorrent.TorrentFilterStalled,
			IncludeTrackers: true,
		},
		ReannounceOptions: qbittorrent.ReannounceOptions{
			Interval:        c.Interval,
			MaxAttempts:     c.MaxAttempts,
			DeleteOnFailure: false,
		},
	}

	if c.Hash != "" {
		opts.TorrentFilterOptions.Filter = qbittorrent.TorrentFilterAll
		opts.TorrentFilterOptions.Hashes = []string{c.Hash}
	} else if c.MaxAttempts == qbittorrent.ReannounceMaxAttempts {
		opts.ReannounceOptions.MaxAttempts = defaultMaxAttemptsDaemon
	}

	return opts
}
