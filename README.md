# qbrr

## Description

qbrr is a CLI tool for reannouncing torrents in qBittorrent with problematic trackers, written in Go using the [github.com/autobrr/go-qbittorrent](https://github.com/autobrr/go-qbittorrent) client.

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
  - [Reannounce hash](#reannounce-hash)
  - [Reannounce daemon](#reannounce-daemon)

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

- `-l, --log-level`: Log level (debug, info, warn, error)

## Usage

### Help

Use the help command to see all available options:

```bash
qbrr --help
```

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
