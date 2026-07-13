# Improvements / Roadmap

This is a living list of potential improvements to `sonosh`, captured from current gaps vs. typical Sonos controller features. Use it as a backlog; we’ll implement items one-by-one.

Legend:
- **Value**: user-facing impact (High/Med/Low)
- **Effort**: estimated implementation effort (S/M/L)
- **Deps**: prerequisites (none / Sonos-linked services / Spotify Web API creds, etc.)

## P0 (high value)

1) **Queue management**
- Value: High | Effort: M | Deps: none
- Add commands:
  - `sonosh queue list --name "<Room>" [--limit N] [--json]`
  - `sonosh queue clear --name "<Room>"`
  - `sonosh queue play --name "<Room>" <index>` (0-based or 1-based, pick one and document)
  - `sonosh queue remove --name "<Room>" <index>`
- Notes:
  - Uses UPnP `ContentDirectory.Browse` (queue container `Q:0`) and `AVTransport.RemoveTrackFromQueue` / `RemoveAllTracksFromQueue` / `Seek TRACK_NR`.
 - Status:
   - Implemented in `0.1.3` (CLI uses 1-based positions).

2) **Better “now playing” metadata**
- Value: High | Effort: M | Deps: none
- Improve `sonosh status`:
  - Parse `TrackMetaData` DIDL and show `title`, `artist`, `album`, `albumArtURI` (when available).
  - Optionally add `sonosh now` as a friendlier alias.
- Notes:
  - Requires DIDL parsing (we can implement a small subset rather than full DIDL).
 - Status:
   - Implemented in `0.1.4` (adds `sonosh now` alias and prints parsed metadata when present).

3) **Group volume + group mute**
- Value: High | Effort: S–M | Deps: none
- Add:
  - `sonosh group volume get|set --name "<AnyMember>" <0-100>`
  - `sonosh group mute get|on|off|toggle --name "<AnyMember>"`
- Notes:
  - Uses `GroupRenderingControl` service (similar to SoCo patterns).
 - Status:
   - Implemented in `0.1.5`.

4) **Presets / scenes**
- Value: High | Effort: L | Deps: none
- Add:
  - `sonosh scene save <name>`: capture grouping + volumes (+ optionally what’s playing)
  - `sonosh scene apply <name>`
  - `sonosh scene list`
- Notes:
  - Needs a config store (file under `~/.config/sonoscli` or similar).
 - Status:
   - Implemented in `0.1.6` (grouping + per-room volume/mute; playback state not captured yet).

## P1 (nice-to-have)

5) **Music sources (Sonos Favorites)**
- Value: Med–High | Effort: M | Deps: favorites must exist
- Add:
  - `sonosh favorites list [--json]`
  - `sonosh favorites open --name "<Room>" "<favorite title>"` (or by index)
- Notes:
  - Favorites are available via `ContentDirectory.Browse` (e.g. `FV:2`), and often include metadata needed to play.
 - Status:
   - Implemented in `0.1.7`.

6) **Music sources (radio / TuneIn / URI play)**
- Value: Med | Effort: M | Deps: depends on source
- Add:
  - `sonosh play-uri --name "<Room>" "<uri>" [--title "..."] [--radio]`
  - `sonosh linein --name "<Room>" [--from "<RoomWithLineIn>"]`
  - `sonosh tv --name "<Room>"`
 - Status:
   - Implemented in `0.1.8`.

7) **Grouping ergonomics**
- Value: Med | Effort: S–M | Deps: none
- Improve:
  - `group join` should accept fuzzy matching and show suggestions on ambiguity.
  - `party` mode: join all speakers to a target.
  - `group dissolve`: unjoin all members of a group.
 - Status:
   - Implemented in `0.1.9`.

8) **Output formats**
- Value: Med | Effort: S | Deps: none
- Add:
  - `--format json|tsv|plain` (or expand `--json` to a more general format flag)
  - Consistent machine-readable JSON across all commands.
 - Status:
   - Implemented in `0.1.10` (`--json` retained as a deprecated alias).

## P2 (advanced / optional)

9) **Event subscriptions + watch mode**
- Value: Med | Effort: L | Deps: network accessibility to event listener port
- Add:
  - `sonosh watch --name "<Room>"` (stream live changes: track/volume/transport)
- Notes:
  - Requires UPnP eventing server on the CLI machine and subscriptions to `AVTransport`/`RenderingControl`.
 - Status:
   - Implemented in `0.1.11` (prints events as they arrive; `--format json` uses one JSON object per line).

10) **Sonos-side music-service browsing/search**
- Value: High | Effort: L | Deps: music service linked in Sonos
- Goal:
  - Search/browse via Sonos (SMAPI) so you can find Spotify content without Spotify Web API credentials.
- Notes:
  - This is more complex and service-dependent.
 - Status:
   - Implemented in `0.1.12` as `sonosh smapi ...` (services list + DeviceLink/AppLink auth + search).

11) **Credential/config management**
- Value: Med | Effort: M | Deps: none
- Add:
  - `sonosh config set spotify.client_id ...`
  - Store secrets in Keychain (macOS) or an encrypted local store; fallback to env vars.

## What needs the Spotify Web API?

- Required:
  - `sonosh search spotify ...` (human query → IDs/URIs)
  - Rich metadata (covers/artist lists) even when Sonos doesn’t provide it cleanly
- Not required:
  - `sonosh open/enqueue` for Spotify when you already have a Spotify URI/share link and Spotify is linked in the Sonos app.
