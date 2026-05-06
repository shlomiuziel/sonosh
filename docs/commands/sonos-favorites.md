---
title: sonos favorites
description: List Sonos Favorites and play one by index or title.
---

# `sonos favorites`

Lists and plays Sonos Favorites — the `FV:2` container in `ContentDirectory`.

```
sonos favorites list [--start N] [--limit N]
sonos favorites open --name "<Room>" [<title>] [--index <n>]
```

## `sonos favorites list`

```bash
sonos favorites list
sonos favorites list --limit 10
sonos favorites list --format json | jq -r '.[].title'
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--start int` | `0` | Starting index (0-based). |
| `--limit int` | `50` | Max results. |

## `sonos favorites open`

Plays a favorite by exact title (case-insensitive) or by 1-based index from `sonos favorites list`.

```bash
sonos favorites open --name "Kitchen" "Morning Coffee"
sonos favorites open --name "Kitchen" --index 3
```

## How it works

- `list` calls `ContentDirectory.Browse(ObjectID=FV:2)` and parses the DIDL-Lite favorites.
- `open` resolves the favorite, sets `AVTransport.SetAVTransportURI` with the favorite's URI + metadata, then calls `Play`.
