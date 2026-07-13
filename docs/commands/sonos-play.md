---
title: sonosh play
description: Resume playback on a room, or search-and-play via Spotify.
---

# `sonosh play`

Sends `AVTransport.Play` to the group coordinator. Has one subcommand for search-and-play through Sonos.

## Synopsis

```
sonosh play --name "<Room>"
sonosh play-url --name "<Room>" <url> [flags]
sonosh play-url --name "<Room>" --playlist-limit 10 <playlist-url>
sonosh play spotify --name "<Room>" <query> [flags]
sonosh play youtube --name "<Room>" <url> [flags]
```

## Examples

```bash
sonosh play --name "Kitchen"
sonosh play-url --name "Kitchen" "https://www.youtube.com/watch?v=-n_rdQIVahw"
sonosh play-url --name "Kitchen" "https://music.youtube.com/playlist?list=PL..."
sonosh play spotify --name "Kitchen" --category albums "kind of blue"
sonosh play youtube --name "Kitchen" "https://www.youtube.com/watch?v=-n_rdQIVahw"
```

## Subcommands

- [`sonosh play-url`](sonos-play-url.md) — stream YouTube, YouTube Music playlists, podcasts, radio, and other URLs through a Sonos-safe local proxy.
- [`sonosh play spotify`](sonos-play-spotify.md) — search Spotify via SMAPI and play the top result.
- [`sonosh play youtube`](sonos-play-youtube.md) — resolve a YouTube URL with `yt-dlp` and play the direct audio stream.

## How it works

`sonosh play` resolves the target's coordinator and calls `AVTransport.Play(InstanceID=0, Speed=1)`. There must already be something on the queue / current URI; if not, the speaker stays silent.

To start a fresh source, use [`sonosh open`](sonos-open.md) (Spotify URI), [`sonosh play-url`](sonos-play-url.md) (web audio), [`sonosh play-uri`](sonos-play-uri.md) (exact URI), or [`sonosh favorites open`](sonos-favorites.md).
