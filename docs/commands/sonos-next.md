---
title: sonos next
description: Skip to the next track in the queue.
---

# `sonos next`

Sends `AVTransport.Next` to the group coordinator.

## Synopsis

```
sonos next --name "<Room>"
```

## Examples

```bash
sonos next --name "Kitchen"
```

## Notes

- Streams (radio, TV) usually don't support `next` — the call is a no-op or returns a fault.
- For Sonos queue / Spotify content, `next` advances by one track.
