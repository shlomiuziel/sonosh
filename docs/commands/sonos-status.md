---
title: sonosh status
description: Print current playback state of a room — transport, track, time, volume, mute.
---

# `sonosh status`

Prints coordinator status: transport state, track URI, position, volume, mute. Parses `TrackMetaData` when available to show title / artist / album / album art. Aliased as `sonosh now`.

## Synopsis

```
sonosh status --name "<Room>" [--format plain|json|tsv]
sonosh now    --name "<Room>"
```

## Examples

```bash
sonosh status --name "Kitchen"
sonosh now    --name "Kitchen"
sonosh status --name "Kitchen" --format json | jq -r .track.title
sonosh status --ip 10.0.0.42
```

## What you get

In `plain`:

- transport state (PLAYING / PAUSED_PLAYBACK / STOPPED / TRANSITIONING)
- current track title / artist / album (when present)
- track position / duration
- volume + mute

In `json`, the same fields with stable keys.

## How it works

- Resolves the target's group coordinator (status reflects the group, not a satellite).
- Calls `AVTransport.GetPositionInfo`, `AVTransport.GetTransportInfo`, `RenderingControl.GetVolume`, `RenderingControl.GetMute` on the coordinator.
- Decodes the `TrackMetaData` DIDL-Lite XML for human-readable track info.

For continuous updates, prefer [`sonosh watch`](sonos-watch.md) over polling.
