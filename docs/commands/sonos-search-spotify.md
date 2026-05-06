---
title: sonos search spotify
description: Search Spotify directly via the Spotify Web API (client credentials) and optionally play / enqueue the result on Sonos.
---

# `sonos search spotify`

Searches Spotify via the Spotify Web API (client credentials). Requires `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` (or the matching flags). Prints Spotify URIs you can hand to [`sonos open`](sonos-open.md) / [`sonos enqueue`](sonos-enqueue.md), or have the command play directly.

## Synopsis

```
sonos search spotify <query> [flags]
```

## Flags

| Flag | Default | What it does |
| --- | --- | --- |
| `--type string` | `track` | `track`, `album`, `playlist`, `show`, `episode`. |
| `--limit int` | `10` | Max results (1–50). |
| `--market string` | — | Optional market (e.g. `US`); empty = global. |
| `--index int` | `1` | Which search result to use with `--open` / `--enqueue` (1-based). |
| `--open` | off | Open the selected result on Sonos (requires `--name` / `--ip`). |
| `--enqueue` | off | Enqueue the selected result on Sonos (requires `--name` / `--ip`). |
| `--client-id string` | — | Spotify Web API client id (or env `SPOTIFY_CLIENT_ID`). |
| `--client-secret string` | — | Spotify Web API client secret (or env `SPOTIFY_CLIENT_SECRET`). |

## Examples

```bash
export SPOTIFY_CLIENT_ID=…
export SPOTIFY_CLIENT_SECRET=…

# Just print URIs
sonos search spotify "miles davis" --type album --limit 5

# Pick result #2 and play on Sonos
sonos search spotify "blue train" --type album --index 2 --open --name "Kitchen"

# Enqueue without playing
sonos search spotify "deep focus" --type playlist --enqueue --name "Kitchen"
```

## When to use this vs SMAPI

Prefer [`sonos play spotify`](sonos-play-spotify.md) (SMAPI) if you don't need Spotify-specific filters — it works with the credentials Sonos already has.

Reach for `search spotify` when you specifically need:

- Market-specific filtering
- `show` / `episode` (podcasts)
- Spotify-side scoring rather than Sonos-side scoring
