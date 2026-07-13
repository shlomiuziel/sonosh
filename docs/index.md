---
title: sonosh
description: A modern Go CLI to discover, group, and control Sonos speakers over your local network — built for terminals, scripts, automations, and coding agents.
permalink: /
---

# sonosh

`sonosh` is a single binary that talks to your Sonos system the way the Sonos app does — over the local network, using UPnP/SOAP — but from the terminal.

## What you can do

- **Discover** every speaker reliably (SSDP + topology + subnet fallback).
- **Control playback** on any room — play, pause, stop, next, prev, line-in, TV.
- **Group rooms** — join, unjoin, dissolve, party-mode, solo, group volume/mute.
- **Manage the queue** — list, play a specific entry, remove, clear.
- **Open Spotify links** without Spotify credentials (uses the Spotify service you already linked in the Sonos app).
- **Search** linked services via Sonos SMAPI, or Spotify directly via Spotify Web API.
- **Play web audio** — proxy YouTube, YouTube Music playlists, podcasts, radio streams, and other `yt-dlp` pages through a Sonos-safe local MP3 stream.
- **Save scenes** — capture grouping + per-room volume/mute and re-apply later.
- **Watch live events** — subscribe to AVTransport / RenderingControl and stream changes.
- **Pipe to JSON / TSV** for scripts and dashboards.

## Why a CLI?

Sonos's IP API is rich, stable, and undocumented in places. The official apps are great for everyday use but painful for automations, kiosks, and AI agents. `sonosh` treats your speakers as plain HTTP devices on port `1400` and exposes the most useful actions as composable subcommands.

This is not an official Sonos project.

## At a glance

```bash
brew install shlomiuziel/tap/sonosh

sonosh discover
sonosh config set defaultRoom "Kitchen"
sonosh config set defaultTimeout 15s
sonosh status --name "Kitchen"
sonosh open --name "Kitchen" "https://open.spotify.com/track/6NmXV4o6bmp704aPGyTVVG"
sonosh play-url --name "Kitchen" "https://music.youtube.com/playlist?list=PL..."
sonosh group party --to "Living Room"
sonosh scene save evening
sonosh watch --name "Kitchen"
```

Pick a starting point:

- New here? Read the [Quickstart](quickstart.md).
- Looking for a specific subcommand? See the [Command Reference](commands/).
- Want to know how it works under the hood? See [Architecture](architecture.md) and [Discovery](discovery.md).
