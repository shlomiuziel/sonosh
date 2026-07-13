---
title: sonosh favorites
description: List Sonos Favorites and play one by index or title.
---

# `sonosh favorites`

Lists and plays Sonos Favorites — the `FV:2` container in `ContentDirectory`.

```
sonosh favorites list [--start N] [--limit N]
sonosh favorites open --name "<Room>" [<title>] [--index <n>]
```

## `sonosh favorites list`

```bash
sonosh favorites list
sonosh favorites list --limit 10
sonosh favorites list --format json | jq -r '.[].title'
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--start int` | `0` | Starting index (0-based). |
| `--limit int` | `50` | Max results. |

## `sonosh favorites open`

Plays a favorite by exact title (case-insensitive) or by 1-based index from `sonosh favorites list`.

```bash
sonosh favorites open --name "Kitchen" "Morning Coffee"
sonosh favorites open --name "Kitchen" --index 3
```

Tip: for services where direct CLI search/playback is not available, such as YouTube Music, save the album or playlist as a Sonos Favorite in the Sonos app, then open that favorite from `sonosh`.

## How it works

- `list` calls `ContentDirectory.Browse(ObjectID=FV:2)` and parses the DIDL-Lite favorites.
- `open` resolves the favorite. Stream and track favorites are sent directly with `AVTransport.SetAVTransportURI`, then `Play`.
- Container favorites such as service-side albums and playlists use the Sonos queue path: clear the queue, enqueue the container with `AddURIToQueue`, switch playback to the queue, seek to the first enqueued track, then `Play`.
