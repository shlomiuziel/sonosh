---
title: sonosh play spotify
description: Search Spotify via Sonos SMAPI and play the top result — no Spotify credentials needed.
---

# `sonosh play spotify`

Uses Sonos's own SMAPI search (no Spotify Web API credentials) to find Spotify content, then enqueues and plays it on the target room's coordinator. Spotify must already be linked in the Sonos app.

## Synopsis

```
sonosh play spotify <query> --name "<Room>" [flags]
```

## Flags

| Flag | Default | What it does |
| --- | --- | --- |
| `--service string` | `Spotify` | Music service name (as shown in `sonosh smapi services`). |
| `--category string` | `tracks` | SMAPI search category — try `tracks`, `albums`, `playlists`. |
| `--index int` | `0` | Result index to play (0-based). |
| `--enqueue` | off | Only enqueue; don't start playback. |
| `--title string` | — | Title override for the queued item. |

## Examples

```bash
sonosh play spotify --name "Kitchen" --category albums "kind of blue"
sonosh play spotify --name "Kitchen" --category tracks "miles davis"
sonosh play spotify --name "Kitchen" --category playlists "deep focus"

# Pick the second result instead of the top one
sonosh play spotify --name "Kitchen" --category albums --index 1 "blue train"

# Enqueue only
sonosh play spotify --name "Kitchen" --enqueue --category albums "kind of blue"
```

## Prerequisites

For Spotify, you'll likely need to do a one-time DeviceLink the first time:

```bash
sonosh auth smapi begin    --service "Spotify"
sonosh auth smapi complete --service "Spotify" --wait 2m
```

## See also

- [Spotify & SMAPI](../spotify-and-smapi.md)
- [`sonosh search spotify`](sonos-search-spotify.md) — Spotify Web API path.
- [`sonosh smapi search`](sonos-smapi.md) — search-only, no playback.
