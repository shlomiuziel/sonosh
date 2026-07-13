---
title: sonosh open
description: Enqueue a Spotify URI / share link and start playback. The fastest way to play Spotify from the terminal.
---

# `sonosh open`

Adds a Spotify item to the queue using `AVTransport.AddURIToQueue`, then starts playback. No Spotify Web API credentials required — Sonos uses the Spotify account already linked in the Sonos app.

## Synopsis

```
sonosh open <spotify-uri-or-link> --name "<Room>" [--next] [--title "<title>"]
```

## Flags

| Flag | What it does |
| --- | --- |
| `--next` | Insert as the next item (shuffle mode only). |
| `--title string` | Optional display title for the queued item. |

## Examples

```bash
# A track share link
sonosh open --name "Kitchen" "https://open.spotify.com/track/6NmXV4o6bmp704aPGyTVVG"

# Canonical URIs work too
sonosh open --name "Kitchen" spotify:album:0nrRP2bk19rLc0orkWPQk2
sonosh open --name "Kitchen" spotify:playlist:37i9dQZF1DXcBWIGoYBM5M
sonosh open --name "Kitchen" spotify:episode:4rOoJ6Egrf8K2IrywzwOMk
```

## Supported types

`track`, `album`, `playlist`, `show`, `episode`.

## See also

- [`sonosh enqueue`](sonos-enqueue.md) — same as `open` but without `Play`.
- [Spotify & SMAPI](../spotify-and-smapi.md) — picking the right path.
