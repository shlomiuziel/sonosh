---
title: Command Reference
description: Every sonosh subcommand at a glance, with one page per command.
---

# Command Reference

The full surface of `sonosh`. Every page documents the command, its flags, examples, and what it talks to on the speaker.

Global flags apply to every command:

| Flag | Default | What it does |
| --- | --- | --- |
| `--name string` | — | Target speaker by name (matched against topology). |
| `--ip string` | — | Target speaker by IP — skips discovery. |
| `--format string` | `plain` | Output format: `plain`, `json`, or `tsv`. |
| `--timeout duration` | `15s` | Discovery and per-call timeout. |
| `--debug` | off | Print SOAP traces to stderr. |

## Discovery & status

- [`sonosh discover`](sonos-discover.md)
- [`sonosh status`](sonos-status.md) (alias: `now`)
- [`sonosh watch`](sonos-watch.md)

## Playback

- [`sonosh play`](sonos-play.md)
- [`sonosh pause`](sonos-pause.md)
- [`sonosh stop`](sonos-stop.md)
- [`sonosh next`](sonos-next.md)
- [`sonosh prev`](sonos-prev.md)
- [`sonosh play-url`](sonos-play-url.md)
- [`sonosh play-uri`](sonos-play-uri.md)
- [`sonosh play youtube`](sonos-play-youtube.md)
- [`sonosh linein`](sonos-linein.md)
- [`sonosh tv`](sonos-tv.md)

## Volume & mute

- [`sonosh volume`](sonos-volume.md)
- [`sonosh mute`](sonos-mute.md)

## Grouping

- [`sonosh group`](sonos-group.md)

## Queue

- [`sonosh queue`](sonos-queue.md)

## Favorites & scenes

- [`sonosh favorites`](sonos-favorites.md)
- [`sonosh scene`](sonos-scene.md)

## Spotify & SMAPI

- [`sonosh open`](sonos-open.md)
- [`sonosh enqueue`](sonos-enqueue.md)
- [`sonosh play spotify`](sonos-play-spotify.md)
- [`sonosh search spotify`](sonos-search-spotify.md)
- [`sonosh smapi`](sonos-smapi.md)
- [`sonosh auth smapi`](sonos-auth-smapi.md)

## Local config

- [`sonosh config`](sonos-config.md)
