---
title: sonos config
description: Read and write small local defaults stored under your user config directory.
---

# `sonos config`

`sonoscli` keeps a small JSON file with local defaults: a default room name, preferred output format, and default discovery/network timeout.

```
sonos config get  [key]
sonos config set  <key> <value>
sonos config unset <key>
sonos config path
```

## Where it lives

```bash
sonos config path
# → ~/.config/sonoscli/config.json   (Linux)
# → ~/Library/Application Support/sonoscli/config.json   (macOS)
# → %AppData%\sonoscli\config.json   (Windows)
```

## Reading

```bash
sonos config get                      # everything
sonos config get defaultRoom          # one key
sonos config get --format json
```

## Writing

```bash
sonos config set defaultRoom "Kitchen"
sonos config set defaultTimeout 10s
sonos config set format json
sonos config unset defaultRoom
```

## Keys

- `defaultRoom`: room used when `--name` / `--ip` is omitted.
- `defaultTimeout`: discovery/network timeout used when `--timeout` is omitted. Built-in default: `15s`.
- `format`: output format used when `--format` is omitted (`plain`, `json`, or `tsv`).

Snake-case aliases are accepted for compatibility: `default_room`, `default_timeout`.

## Notes

- The file is intentionally small — please go through `sonos config` to write it; the schema is allowed to evolve.
- Sensitive items (e.g. SMAPI tokens) are stored separately from this file.
