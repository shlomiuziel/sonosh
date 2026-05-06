---
title: sonos play-uri
description: Set the current transport URI to anything Sonos can play, then start playback.
---

# `sonos play-uri`

Sets the current transport URI on the group coordinator and starts playback. Use `--radio` to force Sonos's radio-style controls for `http(s)` streams.

## Synopsis

```
sonos play-uri <uri> --name "<Room>" [--radio] [--title "<title>"]
```

## Flags

| Flag | What it does |
| --- | --- |
| `--radio` | Force radio-style playback (no seek / no track) for `http(s)` streams. |
| `--title string` | Display title (used as radio metadata). |

## Examples

```bash
# Internet radio
sonos play-uri --name "Kitchen" --radio --title "BBC R4" "http://stream.live.vc.bbcmedia.co.uk/bbc_radio_fourfm"

# A direct file URL Sonos can reach
sonos play-uri --name "Kitchen" "http://10.0.0.10/audio/test.mp3"

# A Sonos-specific scheme you already have
sonos play-uri --name "Kitchen" "x-rincon-stream:RINCON_XXXX"
```

## When to use what

- Have a `spotify:` URI or share link → use [`sonos open`](sonos-open.md).
- Have a Sonos Favorite → use [`sonos favorites open`](sonos-favorites.md).
- Have any other URI Sonos can play (radio, raw file URL, `x-rincon-…`) → `sonos play-uri`.

## How it works

Calls `AVTransport.SetAVTransportURI` with the URI (and minimal DIDL-Lite metadata if `--title` is set), then `AVTransport.Play`.
