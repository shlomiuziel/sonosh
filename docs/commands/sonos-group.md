---
title: sonos group
description: Inspect grouping, join/unjoin rooms, party-mode the house, and adjust group volume / mute.
---

# `sonos group`

Inspect and control grouping. Sub-commands:

```
sonos group status
sonos group join     --name "<Member>"  --to "<Coordinator>"
sonos group unjoin   --name "<Member>"
sonos group dissolve --name "<AnyInGroup>"
sonos group party    --to "<Coordinator>"
sonos group solo     --name "<Room>"
sonos group volume   get|set …
sonos group mute     on|off|toggle|get
```

## `sonos group status`

Shows current groups and members.

```bash
sonos group status
sonos group status --all       # include invisible / bonded devices
sonos group status --format json
```

## `sonos group join`

Makes the target speaker join the group coordinated by `--to`.

```bash
sonos group join --name "Kitchen" --to "Living Room"
```

## `sonos group unjoin`

Makes the target speaker become a standalone coordinator.

```bash
sonos group unjoin --name "Kitchen"
```

## `sonos group dissolve`

Makes every member of the target's group standalone.

```bash
sonos group dissolve --name "Living Room"
```

## `sonos group party`

Makes every visible speaker join the group coordinated by `--to`. Drops anyone already grouped elsewhere first.

```bash
sonos group party --to "Living Room"
```

## `sonos group solo`

Ungroups every other member of the target's group, leaving the target as a standalone coordinator. The inverse of `party`.

```bash
sonos group solo --name "Office"
```

## `sonos group volume`

Controls the group fader, which scales every member proportionally (0–100).

```bash
sonos group volume get --name "Living Room"
sonos group volume set --name "Living Room" 18
```

## `sonos group mute`

Mutes / unmutes the entire group (every member at once).

```bash
sonos group mute on     --name "Living Room"
sonos group mute off    --name "Living Room"
sonos group mute toggle --name "Living Room"
sonos group mute get    --name "Living Room"
```

## How it works

Joining is implemented by setting the member's transport URI to `x-rincon:RINCON_<coordinator-uuid>`. Unjoin / dissolve issue `BecomeCoordinatorOfStandaloneGroup` to each affected member. Group volume / mute use `GroupRenderingControl` on the coordinator.

## See also

- [Architecture · Topology is the source of truth](../architecture.md#topology-is-the-source-of-truth)
- [Scenes](../scenes.md) — save and restore grouping plus volumes.
