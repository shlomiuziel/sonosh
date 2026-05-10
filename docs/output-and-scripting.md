---
title: Output & Scripting
description: Use sonoscli from shell scripts and dashboards — JSON / TSV output, exit codes, and debug tracing.
---

# Output & Scripting

Every command supports the same global flags so you can drop `sonos` into pipes, cron jobs, dashboards, or AI agents without surprises.

## Output formats

```bash
--format plain   # default — readable in a terminal
--format json    # stable shape, pipe to jq
--format tsv     # one record per line, tab-separated
```

Examples:

```bash
sonos discover --format json | jq '.[] | {name, ip, model}'
sonos status   --format json --name "Kitchen" | jq -r .track.title
sonos queue list --name "Kitchen" --format tsv | awk -F'\t' '{print $1, $3}'
```

JSON shapes are versioned implicitly — additive changes (new fields) won't break consumers; renames go through a deprecation cycle.

## Exit codes

- `0` — success
- non-zero — anything else (network failure, speaker not found, UPnP fault, invalid input)

Errors are always printed to stderr. Stdout stays clean for the requested format, so this is safe in pipelines:

```bash
sonos status --name "Kitchen" --format json > /tmp/state.json || alert
```

## Targeting

Most commands need a target room. Two equivalent ways:

```bash
sonos status --name "Kitchen"
sonos status --ip 10.0.0.42
```

`--ip` is the fastest path because it skips discovery. Use it when you already know the IP (e.g. you cache it from a prior `sonos discover --format json`).

## Timeouts

```bash
sonos status --name "Kitchen" --timeout 10s
```

The default is 15 seconds, which is friendlier to slow Wi-Fi and sleepy speakers. Use a shorter value for scripts that should fail fast, or bump it higher on especially flaky networks.

To make a timeout sticky:

```bash
sonos config set defaultTimeout 20s
```

## Debug tracing

```bash
sonos --debug status --name "Kitchen" 2>trace.log
```

`--debug` writes a structured trace to stderr including SOAP requests/responses (with sensitive fields redacted). Use it when something looks wrong; share it when filing an issue.

## Patterns that work well

**Poll until something changes:**

```bash
while sleep 1; do sonos status --name "Kitchen" --format json | jq -r .transport_state; done
```

**Or use the live event stream instead:**

```bash
sonos watch --name "Kitchen"
```

**Fan out to every room:**

```bash
sonos discover --format json | jq -r '.[].name' | xargs -I{} sonos volume set --name "{}" 18
```

**Save state for later:**

```bash
sonos discover --format json > rooms.json
sonos status   --format json --name "Kitchen" > kitchen.json
```

## Tab completion of room names

After your first `sonos discover`, `--name <Tab>` completes against the cached topology in your shell. See [Install](install.md) for setting up completion scripts.
