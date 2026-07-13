---
title: Quickstart
description: A 5-minute tour of sonosh — discover speakers, control playback, group rooms, save a scene, and play a Spotify link.
---

# Quickstart

This walks through the moves you'll use every day. Replace `Kitchen` / `Living Room` with your actual room names.

## 1. Discover speakers

```bash
sonosh discover
```

You should see one row per visible Sonos zone, with name, model, and IP. JSON works too:

```bash
sonosh discover --format json
```

The first run also caches speaker names so `--name <Tab>` autocompletes.

## Optional: Set Local Defaults

If you usually target one room, make commands shorter:

```bash
sonosh config set defaultRoom "Kitchen"
sonosh config set defaultTimeout 15s
```

`defaultTimeout` changes the fallback used by discovery and speaker calls when `--timeout` is omitted. The built-in default is `15s`.

## 2. Check what's playing

```bash
sonosh status --name "Kitchen"
sonosh now --name "Kitchen"           # alias
sonosh status --name "Kitchen" --format json
```

`status` reports on the **group coordinator** — even if you ask a satellite, you get the truthful playback state.

## 3. Control playback

```bash
sonosh play  --name "Kitchen"
sonosh pause --name "Kitchen"
sonosh next  --name "Kitchen"
sonosh prev  --name "Kitchen"
sonosh volume set --name "Kitchen" 25
sonosh mute toggle --name "Kitchen"
```

## 4. Open a Spotify link without credentials

If Spotify is already linked in the Sonos app, you don't need any Spotify Web API setup:

```bash
sonosh open --name "Kitchen" "https://open.spotify.com/track/6NmXV4o6bmp704aPGyTVVG"
sonosh open --name "Kitchen" spotify:album:0nrRP2bk19rLc0orkWPQk2
sonosh enqueue --name "Kitchen" spotify:playlist:37i9dQZF1DXcBWIGoYBM5M --next
```

## 5. Play a URL or playlist

For YouTube, YouTube Music playlists, podcast links, radio streams, and odd formats, use the local proxy path. It resolves common media pages with `yt-dlp`, pipes `yt-dlp` sources into `ffmpeg`, transcodes to MP3, sends the resolved title/provider to Sonos metadata, and exits when playback ends or goes idle:

```bash
sonosh play-url --name "Kitchen" "https://www.youtube.com/watch?v=-n_rdQIVahw"
sonosh play-url --name "Kitchen" "https://music.youtube.com/playlist?list=PL..."
sonosh play-url --name "Kitchen" --playlist-limit 10 "https://music.youtube.com/playlist?list=PL..."
sonosh play-url --name "Kitchen" "https://example.com/podcast/episode.mp3"
```

Unambiguous YouTube / YouTube Music playlist pages (`?list=…` with no `?v=…`) enqueue every track automatically. Use `--playlist` for ambiguous watch+playlist URLs, or `--no-playlist` when you only want the current video.

## 6. Group rooms

```bash
sonosh group status
sonosh group join   --name "Kitchen" --to "Living Room"
sonosh group party  --to "Living Room"
sonosh group volume set --name "Living Room" 18
sonosh group unjoin --name "Kitchen"
sonosh group dissolve --name "Living Room"
```

## 7. Save and apply scenes

A scene captures grouping plus per-room volume/mute. Save what's good now, restore it later.

```bash
sonosh scene save evening
sonosh scene list
sonosh scene apply evening
```

## 8. Watch live events

```bash
sonosh watch --name "Kitchen"
```

Subscribes to AVTransport + RenderingControl and prints state changes as they arrive (Ctrl+C to stop).

## 9. Search via Sonos (no Spotify credentials)

```bash
sonosh smapi services
sonosh smapi search --service "Spotify" --category tracks "miles davis kind of blue"
sonosh play spotify --name "Kitchen" --category albums "kind of blue"
```

If a service needs a one-time link first:

```bash
sonosh auth smapi begin    --service "Spotify"
sonosh auth smapi complete --service "Spotify" --wait 2m
```

## Where to go next

- The full [Command Reference](commands/) lists every flag.
- [Spotify & SMAPI](spotify-and-smapi.md) explains the two search paths.
- [Architecture](architecture.md) covers the moving parts inside the binary.
