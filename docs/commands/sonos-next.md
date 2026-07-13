---
title: sonosh next
description: Skip to the next track in the queue.
---

# `sonosh next`

Sends `AVTransport.Next` to the group coordinator.

## Synopsis

```
sonosh next --name "<Room>"
```

## Examples

```bash
sonosh next --name "Kitchen"
```

## Notes

- Streams (radio, TV) usually don't support `next` — the call is a no-op or returns a fault.
- For Sonos queue / Spotify content, `next` advances by one track.
