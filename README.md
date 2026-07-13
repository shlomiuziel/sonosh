# sonosh

sonosh is a keyboard-driven terminal UI for Sonos.

It keeps the most common controls close to the keyboard: switch rooms, browse the queue, inspect what is playing, and trigger playback without reaching for a browser or phone.

![sonosh TUI demo](docs/assets/sonosh-demo.gif)

## sonosh at a glance

- room switching
- now playing and queue inspection
- playback controls
- media and playlist browsing
- high-quality album art rendering in the player, with block-image fallbacks when needed
- RTL/bidirectional text rendering for right-to-left metadata in the terminal UI
- macOS media-key binding through the Swift helper for play/pause, next, previous, toggle, and volume up/down (volume keys require Accessibility permission)

## Sonos CLI and backend

The `sonosh` CLI is documented below.

## Install / build

Install with Homebrew:

```bash
brew install shlomiuziel/tap/sonosh
```

This installs the `sonosh` command globally. On macOS, the formula also builds
the Swift media helper and wires it up automatically so playback control keys
and now-playing metadata keep working without extra setup.

Upgrade later:

```bash
brew upgrade shlomiuziel/tap/sonosh
```

First-time Spotify setup:

- If you want Spotify / SMAPI search in `sonosh`, use the inherited `sonosh` CLI: `sonosh auth smapi begin --service "Spotify"` and finish the DeviceLink/AppLink flow.
- Use `sonosh config set defaultRoom "<Room Name>"` and `sonosh config set defaultTimeout 20s` if you want sticky defaults for the TUI and CLI.
- See [`docs/commands/sonos-auth-smapi.md`](docs/commands/sonos-auth-smapi.md) and [`docs/commands/sonos-config.md`](docs/commands/sonos-config.md) for the full command details.

Install from source (Go):

```bash
go install github.com/shlomiuziel/sonosh/cmd/sonosh@latest
sonosh
```

Build `sonosh` locally:

```bash
go build -o sonosh ./cmd/sonosh
./sonosh
```

Build the optional macOS media helper:

```bash
swift build --package-path helpers/macos/sonosh-helper --configuration release
./sonosh --mac-helper-path helpers/macos/sonosh-helper/.build/release/sonosh-macos-helper
```

Build `sonosh` locally:

```bash
go build -o sonosh ./cmd/sonosh
./sonosh --version
```

Useful TUI keys:

- `up`/`down` or `k`/`j`: move selection
- `space`: play/pause
- `s`: stop
- `n` / `p`: next / previous
- `+` / `-`: volume
- `m`: mute
- `/` or `tab`: search
- `enter`: search or play selected result
- `r`: refresh
- `q`: quit

Docker:

```bash
docker build -t sonosh .
docker run --rm --network host -v "$PWD/.sonoscli:/data" sonosh discover
```

Linux containers need `--network host` for SSDP/UPnP discovery. The image includes `ffmpeg`, `yt-dlp`, and `curl`.

## First run

Note: if you installed via Homebrew or `go install`, replace `./sonosh` with `sonosh`.

Discover speakers:

```bash
./sonosh discover
./sonosh discover --format json
./sonosh discover --all # include invisible/bonded devices (advanced)
```

Show room status:

```bash
./sonosh status --name "Kitchen"
./sonosh now --name "Kitchen"
./sonosh status --name "Kitchen" --format json
```

Playback controls:

```bash
./sonosh play --name "Kitchen"
./sonosh pause --name "Kitchen"
./sonosh stop --name "Kitchen"
./sonosh next --name "Kitchen"
./sonosh prev --name "Kitchen"
```

Watch live events:

```bash
./sonosh watch --name "Kitchen"
./sonosh watch --name "Kitchen" --format json
./sonosh watch --name "Kitchen" --format tsv
```

Note: this starts a local callback server for UPnP events; your OS firewall may prompt to allow incoming connections.

## CLI overview

Run `sonosh --help` for the full list. Most commonly used:

- Discovery & status: `discover`, `status`/`now`, `watch`
- Playback controls: `play`, `pause`, `stop`, `next`, `prev`, `open`, `enqueue`, `play-url`, `play-uri`, `linein`, `tv`
- Grouping: `group status`, `group join`, `group unjoin`, `group solo`, `group party`, `group dissolve`
- Queue: `queue list`, `queue play`, `queue remove`, `queue clear`
- Favorites: `favorites list`, `favorites open`
- Scenes: `scene save`, `scene apply`, `scene list`, `scene delete`
- Spotify search: `smapi search` (recommended), optional `search spotify` (Spotify Web API)

## Queue management

List the queue:

```bash
./sonosh queue list --name "Kitchen"
./sonosh queue list --name "Kitchen" --format json
```

Play or remove a queue entry (positions are 1-based):

```bash
./sonosh queue play --name "Kitchen" 1
./sonosh queue remove --name "Kitchen" 3
```

Clear the queue:

```bash
./sonosh queue clear --name "Kitchen"
```

## Scenes

Save the current room layout and volumes as a scene:

```bash
./sonosh scene save "Evening"
```

Restore a saved scene:

```bash
./sonosh scene apply "Evening"
```

Manage saved scenes:

```bash
./sonosh scene list
./sonosh scene delete "Evening"
```

Scenes are stored in your user config dir as `sonoscli/scenes.json` (e.g. `~/.config/sonoscli/scenes.json` on macOS/Linux).

## Favorites

List Sonos Favorites:

```bash
./sonosh favorites list --name "Kitchen"
./sonosh favorites list --name "Kitchen" --format json
```

Play a favorite by index:

```bash
./sonosh favorites open --name "Kitchen" --index 1
```

Or play by title:

```bash
./sonosh favorites open --name "Kitchen" "BBC Radio 6 Music"
```

## Media URLs

Play a direct media URL:

```bash
./sonosh play-uri --name "Kitchen" "https://example.com/stream.mp3"
```

Play a URL through the Sonos-safe local proxy (requires `ffmpeg`; `yt-dlp` for YouTube and media pages):

```bash
./sonosh play-url --name "Kitchen" "https://www.youtube.com/watch?v=-n_rdQIVahw"
./sonosh play-url --name "Kitchen" "https://music.youtube.com/playlist?list=PL..."
./sonosh play-url --name "Kitchen" --playlist-limit 10 "https://music.youtube.com/playlist?list=PL..."
./sonosh play-url --name "Kitchen" "https://example.com/podcast/episode.mp3"
```

Play as radio (useful for live streams):

```bash
./sonosh play-uri --name "Kitchen" --radio --title "My Stream" "https://example.com/live.mp3"
```

Select line-in input:

```bash
./sonosh linein --name "Kitchen" --from "Living Room"
```

Select TV input (soundbar):

```bash
./sonosh tv --name "Living Room"
```

## Room grouping

View current groups:

```bash
./sonosh group status
./sonosh group status --all # include invisible/bonded devices (advanced)
```

Move `Bedroom` into `Living Room`’s group:

```bash
./sonosh group join --name "Bedroom" --to "Living Room"
```

Room targeting supports fuzzy substring matching (and will suggest matches on ambiguity):

```bash
./sonosh group join --name "Off" --to "Bar"     # "Office" joins "Bar"
./sonosh group join --name "Bed" --to "Liv"     # "Bedroom" joins "Living Room"
```

Split a speaker out of its group:

```bash
./sonosh group unjoin --name "Bedroom"
```

Solo a speaker (ungroup its current group so it plays alone):

```bash
./sonosh group solo --name "Office"
```

Party mode (join all visible speakers to a target group):

```bash
./sonosh group party --to "Bar"
```

Remove grouping from the whole group:

```bash
./sonosh group dissolve --name "Living Room"
```

Keep `Office` playing on its own:

```bash
./sonosh group solo --name "Office"
./sonosh open --name "Office" "https://open.spotify.com/album/<id>"
```

Add a speaker back to a group:

```bash
./sonosh group join --name "Office" --to "Bar"
```

Control group volume and mute:

```bash
./sonosh group volume get --name "Living Room"
./sonosh group volume set --name "Living Room" 25

./sonosh group mute get --name "Living Room"
./sonosh group mute toggle --name "Living Room"
```

Control room volume and mute:

```bash
./sonosh volume get --name "Kitchen"
./sonosh volume set --name "Kitchen" 25

./sonosh mute get --name "Kitchen"
./sonosh mute toggle --name "Kitchen"
```

## Room targeting

Target a speaker by:
- `--name "Kitchen"` (Sonos room name)
- `--ip 192.168.0.250` (speaker IP)

Most commands must be sent to the *group coordinator* (the device that owns transport state for the group). `sonosh` resolves the coordinator automatically so commands behave like the Sonos app.

## Spotify search

Search via Sonos (SMAPI; no Spotify Web API credentials):

```bash
./sonosh smapi services
./sonosh smapi categories --service "Spotify"
./sonosh smapi browse --service "Spotify" --id root
./sonosh auth smapi begin --service "Spotify"
# open the printed URL in a browser, link your account, then:
./sonosh auth smapi complete --service "Spotify" --code <linkCode> --wait 5m

./sonosh smapi search --service "Spotify" --category tracks "gareth emery"
./sonosh smapi search --service "Spotify" --category tracks --open --name "Office" "gareth emery"
```

Some AppLink services only return a native app authentication URL and no device-link code. In that case, `auth smapi begin` prints the app URL and notes that `sonosh` cannot complete token storage automatically.

Play from a search query (shortcut for SMAPI search + open):

```bash
./sonosh play spotify --name "Office" "gareth emery"
./sonosh play spotify --name "Office" --category albums "gareth emery"
```

SMAPI tokens are stored under your user config dir as `sonoscli/smapi_tokens.json` (e.g. `~/.config/sonoscli/smapi_tokens.json` on macOS/Linux).

Search via Spotify Web API (prints playable URIs):

```bash
export SPOTIFY_CLIENT_ID="..."
export SPOTIFY_CLIENT_SECRET="..."

./sonosh search spotify --type track --limit 5 "daft punk harder better"
./sonosh search spotify --type playlist "focus"
```

Open the first result on Sonos:

```bash
./sonosh search spotify --open --name "Kitchen" "miles davis so what"
```

Enqueue and play:

```bash
./sonosh open --name "Kitchen" spotify:track:6NmXV4o6bmp704aPGyTVVG
./sonosh open --name "Kitchen" https://open.spotify.com/track/6NmXV4o6bmp704aPGyTVVG
```

Enqueue only:

```bash
./sonosh enqueue --name "Kitchen" spotify:playlist:37i9dQZF1DXcBWIGoYBM5M
```

Notes:
- The enqueue implementation tries Spotify Sonos service numbers `2311` and `3079` for compatibility.
- Use `--title` to override the queue display title for some entries.

## Development

### Makefile

```bash
make fmt
make test
make build
make lint
```

`make lint` requires `golangci-lint`:

```bash
brew install golangci-lint
# or:
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### pnpm helper scripts

This repo includes a minimal `package.json` so you can drive the workflow with `pnpm` (no Node dependencies required):

```bash
pnpm build
pnpm test
pnpm format
pnpm lint

pnpm sonosh -- discover
pnpm sonosh -- status --name "Kitchen"
pnpm sonosh -- open --name "Kitchen" spotify:track:6NmXV4o6bmp704aPGyTVVG
pnpm sonosh -- search spotify "miles davis so what"
pnpm sonosh -- group status
pnpm sonosh -- queue list --name "Kitchen"
```

CI runs: `gofmt` check, `go vet`, `go test`, and `golangci-lint`.

## Global flags

- `--ip <ip>`: target by IP
- `--name <name>`: target by speaker name (defaults to `sonosh config defaultRoom` if set)
- `--timeout <duration>`: discovery/network timeout (default `15s`, configurable with `sonosh config set defaultTimeout 10s`)
- `--format plain|json|tsv`: output format (defaults to `sonosh config format` if set)
- `--json`: deprecated alias for `--format json`
- `--debug`: enable detailed trace logs (SSDP/topology/SOAP timings)

## Configuration

Persist small local defaults so repeated commands stay terse:

```bash
./sonosh config get
./sonosh config set defaultRoom "Office"
./sonosh config set defaultTimeout 10s
./sonosh config set format json
./sonosh config unset defaultRoom
```

Supported keys:

- `defaultRoom`: room used when `--name` / `--ip` is omitted.
- `defaultTimeout`: discovery/network timeout used when `--timeout` is omitted. The built-in default is `15s`.
- `format`: default output format (`plain`, `json`, or `tsv`).

Snake-case aliases (`default_room`, `default_timeout`) are accepted too.

## Troubleshooting

- `discover` is empty:
  - Some networks block multicast/SSDP; `sonosh` falls back to scanning local /24 subnets for port `1400` and then uses Sonos topology to list all rooms.
  - Ensure Wi‑Fi client isolation is off and you’re on the same LAN/subnet.
- Discovery is slow or flaky:
  - The default timeout is `15s`. Use `sonosh config set defaultTimeout 20s` to make a longer timeout sticky, or pass `--timeout 5s` in scripts that should fail fast.
  - Run `sonosh --debug discover` to see whether SSDP multicast is timing out and whether topology calls are slow.
- Discovery / SOAP calls hang or time out on your network:
  - `sonosh` retries local Sonos HTTP/SOAP calls via `curl` as a workaround for some network/firmware quirks.
- Commands fail with UPnP/SOAP errors:
  - Verify you can reach `http://<speaker-ip>:1400/` from this machine.
  - Try targeting by `--name` (it resolves the coordinator).
- Spotify enqueue fails:
  - Confirm Spotify is linked and playable in the Sonos app.
  - Some systems behave differently per firmware/service configuration.

## Inspiration / references

This project was informed by the Sonos control ecosystem and the SoCo Python library:

```text
https://github.com/SoCo/SoCo
```

## Design doc

See [`docs/spec.md`](docs/spec.md).

## License

MIT License. See [`LICENSE`](LICENSE).
