---
title: sonosh queue
description: List, play, remove, and clear entries in the Sonos queue (Q:0).
---

# `sonosh queue`

Manage the playback queue. Sonos exposes the queue as the `Q:0` container in `ContentDirectory`; this command surfaces the four most useful operations.

```
sonosh queue list   --name "<Room>" [--start N] [--limit N]
sonosh queue play   --name "<Room>" <pos>
sonosh queue remove --name "<Room>" <pos>
sonosh queue clear  --name "<Room>"
```

Positions are **1-based** to match how the Sonos app numbers rows.

## `sonosh queue list`

```bash
sonosh queue list --name "Kitchen"
sonosh queue list --name "Kitchen" --limit 10
sonosh queue list --name "Kitchen" --start 20 --limit 10
sonosh queue list --name "Kitchen" --format json
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--start int` | `0` | Starting index (0-based for `Browse`). |
| `--limit int` | `50` | Max results to return. |

## `sonosh queue play`

Jumps to a specific entry by position (1-based) and starts playback.

```bash
sonosh queue play --name "Kitchen" 1
sonosh queue play --name "Kitchen" 12
```

## `sonosh queue remove`

Removes the entry at a position (1-based).

```bash
sonosh queue remove --name "Kitchen" 3
```

## `sonosh queue clear`

Empties the queue.

```bash
sonosh queue clear --name "Kitchen"
```

## How it works

- `list` calls `ContentDirectory.Browse(ObjectID=Q:0, BrowseFlag=BrowseDirectChildren)` and parses the DIDL-Lite XML.
- `play` calls `AVTransport.Seek(Unit=TRACK_NR, Target=<pos>)` followed by `Play`.
- `remove` calls `AVTransport.RemoveTrackFromQueue(ObjectID=Q:0/<pos>)`.
- `clear` calls `AVTransport.RemoveAllTracksFromQueue`.
