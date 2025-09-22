# qbrr

## Description

qbrr is a CLI tool for reannouncing torrents in qBittorrent with problematic trackers, written in Go using the [github.com/autobrr/go-qbittorrent](https://github.com/autobrr/go-qbittorrent) client.

## Why another reannouncer?

- **Tracker-friendly:** Unlike other tools that blindly reannounce torrents at fixed intervals, `qbrr` is *nice* to torrent trackers. It waits for the torrent to make its initial contact with the tracker and checks the status. Only if the tracker is problematic does `qbrr` perform a reannounce. This minimizes unnecessary tracker requests and helps avoid bans or rate-limiting.

- **Scalable concurrency:** `qbrr` uses Go’s `sync.WaitGroup` to process multiple torrents concurrently. This means it can efficiently handle thousands of torrents at once, ensuring no announcements are missed even for large clients running in daemon mode.

## Table of contents

- [Description](#description)
- [Why another reannouncer?](#why-another-reannouncer)
- [Table of contents](#table-of-contents)
- [Installation](#installation)
  - [Docker image](#docker-image)
  - [Building](#building)
- [Configuration](#configuration)
- [Usage](#usage)
  - [Reannounce hash](#reannounce-hash)
  - [Reannounce daemon](#reannounce-daemon)
  - [Help](#help)

## Installation

### Docker image

Run a container with access to host network:

```bash
docker run -it --rm --network host ghcr.io/buroa/qbrr:latest --help
```

### Building

```bash
git clone https://github.com/buroa/qbrr.git && cd qbrr
docker build -t qbrr:latest --pull .
docker run -it --rm --network host qbrr:latest --help
```

## Configuration

### Connection Settings

You can specify qBittorrent connection details using environment variables:

- `QBITTORRENT_HOST`
- `QBITTORRENT_USERNAME`
- `QBITTORRENT_PASSWORD`

### Global Options

- `--log-level`: Log level (debug, info, warn, error)

## Usage

### Reannounce hash

In qBittorrent, check the "Run on torrent added" option and set it to the following command:

```bash
qbrr --hash %I
```

### Reannounce daemon

Automatically reannounce torrents that have problematic trackers:

```bash
qbrr
```

### Help

Use the help command to see all available options:

```bash
qbrr --help
```
