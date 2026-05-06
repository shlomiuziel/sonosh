---
title: sonos linein
description: Play line-in audio from one speaker on another speaker or group.
---

# `sonos linein`

Plays line-in from a source speaker on the target speaker or group. If `--from` is omitted, the target itself is used as the source.

## Synopsis

```
sonos linein --name "<Target>" [--from "<Source>"]
```

## Flags

| Flag | What it does |
| --- | --- |
| `--from string` | Source speaker name or IP that has line-in. Defaults to the target. |

## Examples

```bash
# Play the Connect:Amp's line-in throughout the Kitchen group
sonos linein --name "Kitchen" --from "Connect Amp"

# Same speaker is source and target
sonos linein --name "Office"
```

## How it works

Sets the target's transport URI to `x-rincon-stream:RINCON_<source-uuid>` and starts playback. The source speaker must actually have line-in hardware (Connect, Connect:Amp, Five with line-in cable, Port).
