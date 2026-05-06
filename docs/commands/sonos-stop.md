---
title: sonos stop
description: Stop playback on a room.
---

# `sonos stop`

Sends `AVTransport.Stop` to the group coordinator. Some sources (e.g. TV input) don't support stop, in which case this becomes a no-op.

## Synopsis

```
sonos stop --name "<Room>"
```

## Examples

```bash
sonos stop --name "Kitchen"
```

## Stop vs pause

- `pause` keeps the position; `play` resumes where you left off.
- `stop` resets the position to 0 on most sources.
- For streams (radio / TV), `stop` is generally the right choice; `pause` may not be supported.
