---
title: sonos prev
description: Go to the previous track in the queue.
---

# `sonos prev`

Sends `AVTransport.Previous` to the group coordinator. If the source rejects previous (common for some streams), it restarts the current track instead.

## Synopsis

```
sonos prev --name "<Room>"
```

## Examples

```bash
sonos prev --name "Kitchen"
```

## Behavior detail

The "go back / restart" behavior matches what the Sonos app does when you tap the back button mid-stream — most streams have no concept of a previous item, so restarting the current track is the only sensible action.
