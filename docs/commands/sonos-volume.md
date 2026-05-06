---
title: sonos volume
description: Get or set the per-room volume on the group coordinator.
---

# `sonos volume`

Controls `RenderingControl` volume on the group coordinator (0–100). For whole-group volume, see [`sonos group volume`](sonos-group.md#sonos-group-volume).

## Synopsis

```
sonos volume get --name "<Room>"
sonos volume set --name "<Room>" <0-100>
```

## Examples

```bash
sonos volume get --name "Kitchen"
sonos volume set --name "Kitchen" 25
sonos volume set --name "Kitchen" 0          # softest
sonos volume set --name "Kitchen" 100        # loudest

# Per-room ramp
for v in 5 10 15 20; do sonos volume set --name "Kitchen" $v; sleep 1; done
```

## Per-room vs group

- `sonos volume` — sets one room's volume; other rooms in the same group are unaffected.
- `sonos group volume` — sets the *group fader*, which proportionally scales every member.

If you have a single ungrouped room, the two commands are equivalent.
