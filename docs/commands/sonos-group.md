---
title: sonosh group
description: Inspect grouping, join/unjoin rooms, party-mode the house, and adjust group volume / mute.
---

# `sonosh group`

Inspect and control grouping. Sub-commands:

```
sonosh group status
sonosh group join     --name "<Member>"  --to "<Coordinator>"
sonosh group unjoin   --name "<Member>"
sonosh group dissolve --name "<AnyInGroup>"
sonosh group party    --to "<Coordinator>"
sonosh group solo     --name "<Room>"
sonosh group volume   get|set …
sonosh group mute     on|off|toggle|get
```

## `sonosh group status`

Shows current groups and members.

```bash
sonosh group status
sonosh group status --all       # include invisible / bonded devices
sonosh group status --format json
```

## `sonosh group join`

Makes the target speaker join the group coordinated by `--to`.

```bash
sonosh group join --name "Kitchen" --to "Living Room"
```

## `sonosh group unjoin`

Makes the target speaker become a standalone coordinator.

```bash
sonosh group unjoin --name "Kitchen"
```

## `sonosh group dissolve`

Makes every member of the target's group standalone.

```bash
sonosh group dissolve --name "Living Room"
```

## `sonosh group party`

Makes every visible speaker join the group coordinated by `--to`. Drops anyone already grouped elsewhere first.

```bash
sonosh group party --to "Living Room"
```

## `sonosh group solo`

Ungroups every other member of the target's group, leaving the target as a standalone coordinator. The inverse of `party`.

```bash
sonosh group solo --name "Office"
```

## `sonosh group volume`

Controls the group fader, which scales every member proportionally (0–100).

```bash
sonosh group volume get --name "Living Room"
sonosh group volume set --name "Living Room" 18
```

## `sonosh group mute`

Mutes / unmutes the entire group (every member at once).

```bash
sonosh group mute on     --name "Living Room"
sonosh group mute off    --name "Living Room"
sonosh group mute toggle --name "Living Room"
sonosh group mute get    --name "Living Room"
```

## How it works

Joining is implemented by setting the member's transport URI to `x-rincon:RINCON_<coordinator-uuid>`. Unjoin / dissolve issue `BecomeCoordinatorOfStandaloneGroup` to each affected member. Group volume / mute use `GroupRenderingControl` on the coordinator.

## See also

- [Architecture · Topology is the source of truth](../architecture.md#topology-is-the-source-of-truth)
- [Scenes](../scenes.md) — save and restore grouping plus volumes.
