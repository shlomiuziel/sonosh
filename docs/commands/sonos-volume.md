---
title: sonosh volume
description: Get or set the per-room volume on the group coordinator.
---

# `sonosh volume`

Controls `RenderingControl` volume on the group coordinator (0–100). For whole-group volume, see [`sonosh group volume`](sonos-group.md#sonos-group-volume).

## Synopsis

```
sonosh volume get --name "<Room>"
sonosh volume set --name "<Room>" <0-100>
```

## Examples

```bash
sonosh volume get --name "Kitchen"
sonosh volume set --name "Kitchen" 25
sonosh volume set --name "Kitchen" 0          # softest
sonosh volume set --name "Kitchen" 100        # loudest

# Per-room ramp
for v in 5 10 15 20; do sonosh volume set --name "Kitchen" $v; sleep 1; done
```

## Per-room vs group

- `sonosh volume` — sets one room's volume; other rooms in the same group are unaffected.
- `sonosh group volume` — sets the *group fader*, which proportionally scales every member.

If you have a single ungrouped room, the two commands are equivalent.
