---
title: sonos auth smapi
description: One-time DeviceLink / AppLink flow to authenticate a Sonos music service for SMAPI search.
---

# `sonos auth smapi`

Some music services (Spotify is the notable one) require a one-time DeviceLink/AppLink handshake before SMAPI search works — even when the service is already playable in the Sonos app.

```
sonos auth smapi begin    --service "<Service>"
sonos auth smapi complete --service "<Service>" [--code <code>] [--wait <duration>]
```

## `sonos auth smapi begin`

Starts the linking flow. DeviceLink services print a URL and link code. Some AppLink services only return a native app URL; in that case `sonoscli` reports the URL and explains that it cannot complete token storage automatically.

```bash
sonos auth smapi begin --service "Spotify"
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--service string` | `Spotify` | Music service name. |

## `sonos auth smapi complete`

Completes the flow and stores tokens locally. Pass `--wait` to poll the service until linking is acknowledged.

```bash
sonos auth smapi complete --service "Spotify" --wait 2m
```

| Flag | Default | What it does |
| --- | --- | --- |
| `--service string` | `Spotify` | Service name. |
| `--code string` | — | Link code from `begin` when the service returns a device-link flow. |
| `--link-device-id string` | — | Optional link device id from `begin` (rare). |
| `--wait duration` | `0` | Wait up to this duration for linking to complete (polls). |

## End-to-end

```bash
sonos auth smapi begin    --service "Spotify"
# … browser flow …
sonos auth smapi complete --service "Spotify" --wait 2m
sonos smapi search        --service "Spotify" --category tracks "miles davis"
```

Tokens are stored locally so you only do this once per machine, per service.

Native AppLink-only services, such as Apple Music in some regions, may return an app URL without a device-link code. Those flows must be completed in the native service/Sonos app; `sonoscli` cannot store SMAPI tokens unless the service exposes a device-link code.
