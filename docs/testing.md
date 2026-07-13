# Testing

This document is the manual + semi-automated test plan for `sonosh`.

Goals:
- Catch regressions in discovery/topology, grouping, and playback control.
- Provide a repeatable “does this work on my network?” checklist.

## Prereqs

- Go `1.22+`
- `golangci-lint` installed (for `make lint` / `pnpm lint`)
- `ffmpeg` and `yt-dlp` installed for `play-url` checks
- Sonos speakers reachable on the local network (UDP SSDP + TCP 1400)

## Quick checks (automated)

Run from repo root:

- `pnpm -s build`
- `pnpm -s format:check`
- `pnpm -s test`
- `pnpm -s lint`
- `make ci`
- Optional: `go test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out | tail -n 1`

Expected:
- All commands exit `0`
- CI should match `.github/workflows/ci.yml` (`gofmt`, coverage, `golangci-lint`, race tests, `go vet`)
- CI enforces a minimum total coverage of `75%` (statement coverage across `./...`) and a focused `85%` floor for the stream proxy package.

## Live network test plan (manual)

Notes:
- Some tests change grouping and playback. Prefer using a “test room” (e.g. `Office`).
- Default timeout is 15s. Use a higher `--timeout` if your network is very slow.

### 1) CLI basics

- `sonosh --version` prints the version (matches `internal/cli/version.go`)
- `sonosh --help` works
- `sonosh <cmd> --help` works for core commands (`discover`, `status`, `group`, `open`, `scene`, `smapi`)

### 2) Discovery + topology

### 2.5) Discovery (advanced)

- `sonosh discover --all` includes invisible/bonded devices (useful for debugging)
- `sonosh discover --format json` prints structured results
- `sonosh discover --format tsv` prints tab-separated output


- `sonosh discover --timeout 6s`
  - Expected: prints all visible rooms (name, IP, UUID)
- `sonosh group status`
  - Expected: prints group coordinators + members
- `sonosh status --name "<room>"`
  - Expected: prints speaker metadata + playback state

Regression checks:
- If SSDP multicast fails, discovery should fall back to subnet scan + topology and still find rooms.

### 3) Volume + mute

### 3.5) Config defaults

- `sonosh config path` prints where config is stored
- `sonosh config set defaultRoom "Office"` then run a command without `--name`/`--ip`:
  - `sonosh volume get` (should target the default room)
- `sonosh config set defaultTimeout 10s`, then `sonosh --help` should show `--timeout ... (default 10s)`
- `sonosh config unset defaultTimeout`, then `sonosh --help` should show `--timeout ... (default 15s)`
- `sonosh config unset defaultRoom` then run `sonosh volume get` (should error and ask for `--name/--ip`)


Pick a room:

- `sonosh volume get --name "<room>"`
- `sonosh mute get --name "<room>"`
- `sonosh volume set --name "<room>" 12`
- `sonosh mute on --name "<room>"`
- `sonosh mute off --name "<room>"`

Expected:
- Values change immediately and `sonosh status` reflects the new values.

### 4) Grouping controls

### 4.5) Group volume/mute

Create a small temporary group (recommended: join `Pantry` to `Office`) and validate group-wide controls:

- `sonosh group join --name Pantry --to Office`
- `sonosh group volume get --name Office`
- `sonosh group volume set --name Office 18`
- `sonosh group mute toggle --name Office` (twice to return to original)
- `sonosh group dissolve --name Office` (splits the test group)


Pick a coordinator room and a second room:

- `sonosh group join --name "<member>" --to "<coordinator>"`
- `sonosh group status` shows member under coordinator
- `sonosh group unjoin --name "<member>"`
- `sonosh group status` shows member as its own coordinator

Optional:
- `sonosh group party --name "<coordinator>"` (joins all visible rooms)
- `sonosh group dissolve --name "<coordinator>"` (ungroups all members)
- `sonosh group solo --name "<room>"` (ensures it plays by itself)

### 5) Inputs (TV/Line-in)

TV input (soundbars/home theater):
- Ensure the target is the *soundbar* (e.g. `Living Room`) and it is a standalone coordinator:
  - `sonosh group solo --name "<soundbar room>"`
- `sonosh tv --name "<soundbar room>"`
- `sonosh status --name "<soundbar room>"` should show a URI like `x-sonos-htastream:<UUID>:spdif`

Line-in (devices with analog-in, e.g. Sonos Five):
- `sonosh linein --name "<room>"` (defaults `--from` to the same room)
- `sonosh status --name "<room>"` should show a URI like `x-rincon-stream:<UUID>`

### 6) Spotify playback (no Spotify Web API creds)

This uses Sonos queueing (AVTransport) + Spotify share links.

- `sonosh open --name "<room>" "https://open.spotify.com/track/<id>"`
- `sonosh open --name "<room>" "https://open.spotify.com/album/<id>"`
- `sonosh enqueue --name "<room>" "spotify:track:<id>"`
- `sonosh next --name "<room>"`
- `sonosh pause --name "<room>"`
- `sonosh play --name "<room>"`
- `sonosh stop --name "<room>"`

Expected:
- Playback starts, track info updates in `sonosh status`

### 7) Queue management

- `sonosh queue list --name "<room>"`
- `sonosh queue clear --name "<room>"`

Expected:
- List shows items when queue is in use
- Clear empties the queue

### 8) Scenes (grouping + volume presets)

Scenes should only include *visible* rooms (not bonded satellites/subs).

- `sonosh scene save __tmp_test`
- `sonosh scene apply __tmp_test`
- `sonosh scene list`
- `sonosh scene delete __tmp_test`

Expected:
- No `soap http 500` errors on systems with home theater / stereo bonded devices.

### 9) Sonos Favorites

- `sonosh favorites list --name "<room>" --timeout 10s`
- `sonosh favorites open --name "<room>" --index 1`

Expected:
- Lists favorites; plays selected item (if any exist).

### 10) SMAPI (Sonos music services)

SMAPI is Sonos-side browsing/search for linked services (e.g. Spotify). It may require a one-time DeviceLink/AppLink auth flow.

- `sonosh smapi services`
- `sonosh smapi categories --service "Spotify"`
- `sonosh smapi search --service "Spotify" --category tracks "miles davis"`

If auth is required:
- `sonosh auth smapi begin --service "Spotify"` (follow the `regUrl` and link code)
- `sonosh auth smapi complete --service "Spotify" --code "<linkCode>" --wait 2m`
- Some AppLink services, such as Apple Music, may return a native app URL without a device-link code; verify the command reports that clearly instead of printing empty completion instructions.

Expected:
- Categories show at least `tracks`, `albums`, `artists`, `playlists` for Spotify.
- Search returns results after auth is completed.

### 11) URL streaming proxy

Use a room you can interrupt. The command starts a short-lived local daemon and returns after Sonos accepts the stream.

- `sonosh play-url --name "<room>" "https://www.youtube.com/watch?v=-n_rdQIVahw"`
- `sonosh play-url --name "<room>" "https://example.com/podcast/episode.mp3"`
- `sonosh play-url --name "<room>" --playlist-limit 2 "https://music.youtube.com/playlist?list=PL..."`

Expected:
- Single URLs start playback through a local `http://<local-ip>:<port>/Sonos%20CLI.mp3` proxy.
- YouTube HLS-only videos play through the `yt-dlp` to `ffmpeg` pipeline instead of failing after the command returns.
- Playlist mode clears the queue, enqueues one local MP3 URL per track, and starts at track 1.
- The proxy exits after EOF or after the idle timeout when playback is abandoned.

### 12) Event watching (manual)

- `sonosh watch --name "<room>" --duration 15s` (or omit `--duration` and Ctrl+C)
- Change volume / skip track in another controller/app.

Expected:
- Events stream in (may take a few seconds after the change); stop with Ctrl+C.

### 13) Shell completions

- `sonosh completion zsh`
- `sonosh completion bash`
- `sonosh completion fish`
- `sonosh completion powershell`

Expected: prints a completion script to stdout.

## Latest run (example record)

Fill this in when doing an end-to-end run.

- Date: `2025-12-13T17:55:22Z`
- Commit SHA: `0253bd1`
- Network: `192.168.0.0/24`
- Discovery result (rooms found): `Bar, Bedroom, Hallway, Kitchen, Living Room, Master Bathroom, Office, Pantry`
- Notes/issues:
  - Verified: `sonosh --version` prints `0.1.27`.
  - Verified: `sonosh discover --timeout 6s` finds all expected rooms; `sonosh discover --all --format tsv` includes bonded/hidden devices; `sonosh discover --format json` prints a JSON array.
  - Verified: Config defaults:
    - `sonosh config set defaultRoom Office` makes `sonosh volume get` work without `--name`/`--ip`.
    - `sonosh config set format tsv` affects default output format; reset to `plain` after.
  - Verified: Spotify playback on Office:
    - `sonosh group solo --name Office` isolates `Office` for playback.
    - `sonosh open --name Office https://open.spotify.com/album/4o9BvaaFDTBLFxzK70GT1E?...` starts playback.
    - `sonosh queue list/play/remove/clear --name Office` works.
  - Verified: Group controls:
    - `sonosh group join --name Pantry --to Office`
    - `sonosh group volume get/set --name Office`, `sonosh group mute toggle --name Office`, `sonosh group dissolve --name Office`
  - Verified: Scenes:
    - `sonosh scene save/apply/list/delete __tmp_test` works.
  - Verified: Favorites:
    - `sonosh favorites list --name Office` and `sonosh favorites open --name Office --index 1` work.
  - Verified: SMAPI / Spotify search:
    - `sonosh smapi services` and `sonosh smapi search --service Spotify --category tracks "miles davis"` work.
    - `sonosh smapi search --open --name Office --index 1 "miles davis"` starts playback.
    - `sonosh smapi browse --service Spotify --id root` works (drill into returned container ids).
  - Verified: `sonosh group unjoin --name Pantry` removes it from the Bar group (restored afterward).
  - Verified: Mute toggle:
    - `sonosh mute toggle --name Office` flips mute state and `sonosh mute get` reflects it.
  - Verified: Play URI (radio):
    - `sonosh play-uri --name Office --radio http://stream.radioparadise.com/mp3-192` starts playback (URI shows `x-rincon-mp3radio://...`).
    - Note: Some public streams may not play (depends on codec/redirects/Sonos support).
  - Verified: Watch:
    - `sonosh watch --name Office --duration 6s --format tsv` reports `volume_master` changes when volume is adjusted during the watch window.
    - `sonosh watch --name Office --duration 4s --format json` prints one JSON object per event line.
  - Verified: Shell completions:
    - `sonosh completion zsh|bash|fish|powershell` prints completion scripts.
  - Verified: Spotify Web API search errors clearly without credentials:
    - `sonosh search spotify "miles davis"` prints a “missing SPOTIFY_CLIENT_ID / SPOTIFY_CLIENT_SECRET” error.
  - Verified: JSON output:
    - `sonosh --format json discover` prints a JSON array.
    - `sonosh --format json queue list --name Office` prints a JSON object with `items`.
  - Verified: `sonosh discover --timeout 10ms` now returns a clear error (non-zero exit) when no speakers are found; `--format json` still prints `[]`.
  - Verified: Group party/dissolve:
    - `sonosh group party --to Bar` joins all rooms into the Bar group.
    - `sonosh group dissolve --name Bar` returns rooms to standalone coordinators.
  - Verified: Enqueue does not start playback:
    - `sonosh stop --name Office` then `sonosh enqueue --name Office spotify:track:...` keeps `State: STOPPED`.
    - `sonosh queue play --name Office 1` starts playback.
  - Verified: Volume boundaries:
    - `sonosh volume set --name Office 0` and `sonosh volume set --name Office 100` work and reflect in `sonosh volume get`.
    - `sonosh group volume set --name Office 0|100` works on a temporary `Office+Pantry` group.
  - Restored original state via `sonosh --timeout 25s scene apply/delete __restore_continued_testing_2025_12_13_b`.
