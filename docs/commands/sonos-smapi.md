---
title: sonosh smapi
description: Browse and search any music service linked in your Sonos app via the SMAPI protocol.
---

# `sonosh smapi`

SMAPI (Sonos Music API) is the same protocol the Sonos app uses to browse and search music services — Spotify, Apple Music, TuneIn, Mixcloud, and friends. No third-party app credentials are needed, but some services require local SMAPI auth before search or playback works.

```
sonosh smapi services
sonosh smapi categories --service "<Service>"
sonosh smapi search     --service "<Service>" --category <cat> <query>
sonosh smapi browse     --service "<Service>" [--id <container>] [--recursive]
```

## `sonosh smapi services`

Lists the Sonos music-service catalog available to the household/region. A listed service is not necessarily authenticated for local SMAPI search yet.

```bash
sonosh smapi services
sonosh smapi services --format json
```

## `sonosh smapi categories`

Lists the searchable categories for one service.

```bash
sonosh smapi categories --service "Spotify"
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--service string` | `Spotify` | Music service name (as shown in `sonosh smapi services`). |

## `sonosh smapi search`

Searches a linked service.

```bash
sonosh smapi search --service "Spotify" --category tracks "miles davis"
sonosh smapi search --service "Spotify" --category albums --limit 25 "kind of blue"

# Pick a result and play it
sonosh smapi search --service "Spotify" --category tracks "miles davis" \
  --open --name "Kitchen" --index 1
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--service string` | `Spotify` | Service name. |
| `--category string` | `tracks` | Category — service-dependent (`tracks`, `albums`, `artists`, `playlists`). |
| `--limit int` | `10` | Max results (1–200, service-dependent). |
| `--index int` | `1` | Result index for `--open` / `--enqueue` (1-based). |
| `--open` | off | Play the selected result. Requires `--name` / `--ip`. |
| `--enqueue` | off | Enqueue the selected result. Requires `--name` / `--ip`. |

## `sonosh smapi browse`

Browse a music service container hierarchically.

```bash
sonosh smapi browse --service "Spotify" --id root
sonosh smapi browse --service "Spotify" --id <some-container-id>
sonosh smapi browse --service "Spotify" --id root --recursive
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--service string` | `Spotify` | Service name. |
| `--id string` | `root` | Container/item id. Use `root` to start. |
| `--limit int` | `50` | Max results. |
| `--recursive` | off | Recursively browse (service-dependent). |
| `--index int` | `1` | Result index for `--open` / `--enqueue`. |
| `--open` | off | Play selected item. |
| `--enqueue` | off | Enqueue selected item. |

## Playback compatibility

For Spotify results, `sonosh` uses the dedicated Spotify queue URI forms. For other SMAPI services, `--open` and `--enqueue` use the generic Sonos queue path with SoCo-compatible metadata. This is required by some AppLink services, including QQ Music and NetEase Cloud Music.

## Authenticating a service

Some services need a one-time DeviceLink/AppLink flow before SMAPI search works. See [`sonosh auth smapi`](sonos-auth-smapi.md).

## See also

- [Spotify & SMAPI](../spotify-and-smapi.md) — picking the right path.
