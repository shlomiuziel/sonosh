---
title: sonos pause
description: Pause playback on a room.
---

# `sonos pause`

Sends `AVTransport.Pause` to the group coordinator.

## Synopsis

```
sonos pause --name "<Room>"
```

## Examples

```bash
sonos pause --name "Kitchen"
sonos pause --ip 10.0.0.42
```

## Notes

- Resolves the coordinator automatically — pausing a satellite still pauses the whole group.
- Some sources (e.g. TV input) don't support pause; the call becomes a no-op.
