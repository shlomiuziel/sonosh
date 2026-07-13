---
title: sonosh tv
description: Switch a soundbar (Beam, Arc, Playbar, Ray, …) to its TV input.
---

# `sonosh tv`

Switches the target speaker / group to the TV input. Only meaningful on home-theater products (Beam, Arc, Playbar, Playbase, Ray).

## Synopsis

```
sonosh tv --name "<Soundbar>"
```

## Examples

```bash
sonosh tv --name "Living Room"
```

## How it works

Sets the transport URI to `x-sonos-htastream:RINCON_<uuid>:spdif` (the local TV input). Has no effect on speakers without HT input.
