---
title: sonosh mute
description: Get, set, or toggle per-room mute on the group coordinator.
---

# `sonosh mute`

Controls `RenderingControl` mute on the group coordinator. For whole-group mute, see [`sonosh group mute`](sonos-group.md#sonos-group-mute).

## Synopsis

```
sonosh mute <on|off|toggle|get> --name "<Room>"
```

## Examples

```bash
sonosh mute on     --name "Kitchen"
sonosh mute off    --name "Kitchen"
sonosh mute toggle --name "Kitchen"
sonosh mute get    --name "Kitchen"
```

## Notes

- `mute` is independent from `volume` — un-muting restores the previous volume; muting doesn't lose the volume value.
- For grouped rooms, you can mute a single member without affecting the rest of the group. To silence the whole group, use [`sonosh group mute on`](sonos-group.md#sonos-group-mute).
