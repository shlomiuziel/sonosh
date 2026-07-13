---
title: sonosh config
description: Read and write small local defaults stored under your user config directory.
---

# `sonosh config`

`sonosh` keeps a small JSON file with local defaults: a default room name, preferred output format, and default discovery/network timeout.

```
sonosh config get  [key]
sonosh config set  <key> <value>
sonosh config unset <key>
sonosh config path
```

## Where it lives

```bash
sonosh config path
# → ~/.config/sonoscli/config.json   (Linux)
# → ~/Library/Application Support/sonoscli/config.json   (macOS)
# → %AppData%\sonoscli\config.json   (Windows)
```

## Reading

```bash
sonosh config get                      # everything
sonosh config get defaultRoom          # one key
sonosh config get --format json
```

## Writing

```bash
sonosh config set defaultRoom "Kitchen"
sonosh config set defaultTimeout 10s
sonosh config set format json
sonosh config unset defaultRoom
```

## Keys

- `defaultRoom`: room used when `--name` / `--ip` is omitted.
- `defaultTimeout`: discovery/network timeout used when `--timeout` is omitted. Built-in default: `15s`.
- `format`: output format used when `--format` is omitted (`plain`, `json`, or `tsv`).

Snake-case aliases are accepted for compatibility: `default_room`, `default_timeout`.

## Notes

- The file is intentionally small — please go through `sonosh config` to write it; the schema is allowed to evolve.
- Sensitive items (e.g. SMAPI tokens) are stored separately from this file.
