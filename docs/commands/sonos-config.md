---
title: sonos config
description: Read and write small local defaults stored under your user config directory.
---

# `sonos config`

`sonoscli` keeps a small JSON file with local defaults — typical example: a default room name so you don't have to pass `--name` every time.

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
sonos config set default_room "Kitchen"
sonos config set defaultTimeout 10s
sonos config unset default_room
```

## Notes

- The file is intentionally small — please go through `sonos config` to write it; the schema is allowed to evolve.
- Supported keys: `defaultRoom`, `defaultTimeout`, `format`. Snake_case aliases are accepted for `default_room` and `default_timeout`.
- Sensitive items (e.g. SMAPI tokens) are stored separately from this file.
