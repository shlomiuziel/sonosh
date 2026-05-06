---
title: sonos mute
description: Get, set, or toggle per-room mute on the group coordinator.
---

# `sonos mute`

Controls `RenderingControl` mute on the group coordinator. For whole-group mute, see [`sonos group mute`](sonos-group.md#sonos-group-mute).

## Synopsis

```
sonos mute <on|off|toggle|get> --name "<Room>"
```

## Examples

```bash
sonos mute on     --name "Kitchen"
sonos mute off    --name "Kitchen"
sonos mute toggle --name "Kitchen"
sonos mute get    --name "Kitchen"
```

## Notes

- `mute` is independent from `volume` — un-muting restores the previous volume; muting doesn't lose the volume value.
- For grouped rooms, you can mute a single member without affecting the rest of the group. To silence the whole group, use [`sonos group mute on`](sonos-group.md#sonos-group-mute).
