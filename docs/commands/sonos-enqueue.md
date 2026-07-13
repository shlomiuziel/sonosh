---
title: sonosh enqueue
description: Enqueue a Spotify URI without starting playback.
---

# `sonosh enqueue`

Same as [`sonosh open`](sonos-open.md) but stops short of pressing play — the item is added to the queue, that's it.

## Synopsis

```
sonosh enqueue <spotify-uri-or-link> --name "<Room>" [--next] [--title "<title>"]
```

## Flags

| Flag | What it does |
| --- | --- |
| `--next` | Insert as the next item (shuffle mode only). |
| `--title string` | Optional display title for the queued item. |

## Examples

```bash
sonosh enqueue --name "Kitchen" spotify:track:6NmXV4o6bmp704aPGyTVVG
sonosh enqueue --name "Kitchen" spotify:album:0nrRP2bk19rLc0orkWPQk2 --next
```

## When to use this vs `open`

- Already playing something and want to extend the queue → `enqueue`.
- Replace what's playing immediately → `open`.
- Want it to play right after the current track → `enqueue --next` (shuffle mode only).
