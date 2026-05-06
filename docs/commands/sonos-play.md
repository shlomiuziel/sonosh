---
title: sonos play
description: Resume playback on a room, or search-and-play via Spotify.
---

# `sonos play`

Sends `AVTransport.Play` to the group coordinator. Has one subcommand for search-and-play through Sonos.

## Synopsis

```
sonos play --name "<Room>"
sonos play spotify --name "<Room>" <query> [flags]
```

## Examples

```bash
sonos play --name "Kitchen"
sonos play spotify --name "Kitchen" --category albums "kind of blue"
```

## Subcommands

- [`sonos play spotify`](sonos-play-spotify.md) — search Spotify via SMAPI and play the top result.

## How it works

`sonos play` resolves the target's coordinator and calls `AVTransport.Play(InstanceID=0, Speed=1)`. There must already be something on the queue / current URI; if not, the speaker stays silent.

To start a fresh source, use [`sonos open`](sonos-open.md) (Spotify URI), [`sonos play-uri`](sonos-play-uri.md) (any URI), or [`sonos favorites open`](sonos-favorites.md).
