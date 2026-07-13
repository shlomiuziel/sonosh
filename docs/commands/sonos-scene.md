---
title: sonosh scene
description: Save, list, apply, and delete scenes — named snapshots of grouping plus per-room volume / mute.
---

# `sonosh scene`

A scene captures the current grouping (coordinators + members) and per-room volume / mute, and can be re-applied later.

```
sonosh scene save   <name>
sonosh scene list
sonosh scene apply  <name> [--only "<Room>"]
sonosh scene delete <name>
```

## `sonosh scene save`

```bash
sonosh scene save evening
sonosh scene save weekday-morning
```

Walks the live topology and writes a JSON snapshot under your config directory.

## `sonosh scene list`

```bash
sonosh scene list
sonosh scene list --format json
```

## `sonosh scene apply`

```bash
sonosh scene apply evening
sonosh scene apply evening --only "Kitchen"   # experimental
```

| Flag | What it does |
| --- | --- |
| `--only string` | Restore only the saved state for one room (experimental). |

Apply is best-effort and idempotent — if a member is offline, that step is skipped and the rest of the scene is still applied.

## `sonosh scene delete`

```bash
sonosh scene delete evening
```

## See also

- [Scenes](../scenes.md) — what's captured, what isn't, and how apply works under the hood.
