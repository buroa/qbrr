# qbr

## Description

qbr is a CLI tool for reannouncing torrents in qBittorrent with problematic trackers, written in Go using the [github.com/autobrr/go-qbittorrent](https://github.com/autobrr/go-qbittorrent) client.

## Features

- **Reannounce**: Automatically reannounce torrents with problematic trackers

## Table of contents

- [Description](#description)
- [Features](#features)
- [Table of contents](#table-of-contents)
- [Installation](#installation)
  - [Docker image](#docker-image)
  - [Building](#building)
- [Configuration](#configuration)
- [Usage](#usage)
  - [Help](#help)
  - [Reannounce](#reannounce)

## Installation

### Docker image

Run a container with access to host network:

```bash
docker run -it --rm --network host ghcr.io/buroa/qbr:latest --help
```

### Building

```bash
git clone https://github.com/buroa/qbr.git && cd qbr
docker build -t qbr:latest --pull .
docker run -it --rm --network host qbr --help
```

## Configuration

### Connection Settings

You can specify qBittorrent connection details using environment variables:

- `QBITTORRENT_HOST`
- `QBITTORRENT_USERNAME`
- `QBITTORRENT_PASSWORD`

### Global Options

- `-l, --log-level`: Log level (debug, info, warn, error)

## Usage

### Help

Use the help command to see all available options:

```bash
qbr --help
```

### Reannounce

Automatically reannounce torrents that have problematic trackers:

- `--max-age`: Maximum age of torrents to reannounce in seconds (default: 3600)
- `--max-retries`: Maximum number of reannounce attempts (default: 18)
- `--interval`: Interval between reannouncements in seconds (default: 5)

```bash
qbr --max-age 7200
```

The reannounce process will:
1. Find torrents with problematic trackers (stalled downloading torrents by default)
2. Automatically reannounce them with configurable retry logic
3. Continue running in a loop with a 5-second interval between cycles
