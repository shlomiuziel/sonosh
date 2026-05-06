---
title: sonos queue
description: List, play, remove, and clear entries in the Sonos queue (Q:0).
---

# `sonos queue`

Manage the playback queue. Sonos exposes the queue as the `Q:0` container in `ContentDirectory`; this command surfaces the four most useful operations.

```
sonos queue list   --name "<Room>" [--start N] [--limit N]
sonos queue play   --name "<Room>" <pos>
sonos queue remove --name "<Room>" <pos>
sonos queue clear  --name "<Room>"
```

Positions are **1-based** to match how the Sonos app numbers rows.

## `sonos queue list`

```bash
sonos queue list --name "Kitchen"
sonos queue list --name "Kitchen" --limit 10
sonos queue list --name "Kitchen" --start 20 --limit 10
sonos queue list --name "Kitchen" --format json
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--start int` | `0` | Starting index (0-based for `Browse`). |
| `--limit int` | `50` | Max results to return. |

## `sonos queue play`

Jumps to a specific entry by position (1-based) and starts playback.

```bash
sonos queue play --name "Kitchen" 1
sonos queue play --name "Kitchen" 12
```

## `sonos queue remove`

Removes the entry at a position (1-based).

```bash
sonos queue remove --name "Kitchen" 3
```

## `sonos queue clear`

Empties the queue.

```bash
sonos queue clear --name "Kitchen"
```

## How it works

- `list` calls `ContentDirectory.Browse(ObjectID=Q:0, BrowseFlag=BrowseDirectChildren)` and parses the DIDL-Lite XML.
- `play` calls `AVTransport.Seek(Unit=TRACK_NR, Target=<pos>)` followed by `Play`.
- `remove` calls `AVTransport.RemoveTrackFromQueue(ObjectID=Q:0/<pos>)`.
- `clear` calls `AVTransport.RemoveAllTracksFromQueue`.
