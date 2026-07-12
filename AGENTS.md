# Repository Guidelines

## Project Structure & Module Organization
`cmd/sonosh` contains the CLI entrypoint. Core application code lives under `internal/`, with the TUI in `internal/tui`, Sonos integrations in `internal/sonos`, and macOS helper integration in `internal/macoshelper`. The Swift helper binary is in `helpers/macos/sonosh-helper`. Docs and site assets live in `docs/`, utility scripts in `scripts/`, and packaging metadata in `Formula/` and `homebrew-tap/`.

## Build, Test, and Development Commands
- `make build`: build the `sonosh` binary locally.
- `make test`: run the Go test suite.
- `make race`: run tests with the race detector.
- `make coverage`: generate `coverage.out`.
- `make fmt` / `make fmt-check`: format code or verify formatting with `gofmt`.
- `make lint`: run `golangci-lint`.
- `make ci`: run the main local CI sequence.
- `swift build --package-path helpers/macos/sonosh-helper --configuration release`: build the macOS helper.

## Coding Style & Naming Conventions
Use standard Go formatting with `gofmt`; CI enforces a clean `gofmt -l .` result. Keep Go code idiomatic: exported names in `CamelCase`, unexported names in `camelCase`, and table-driven tests where they improve clarity. Prefer small focused helpers over broad utility layers. Match existing file naming such as `*_store.go` with paired `*_store_test.go`.

## Testing Guidelines
Tests use Go’s built-in `testing` package. Place tests next to the code they cover and name them `TestXxx`. Run targeted tests while iterating, for example `go test ./internal/tui -run TestModelLoadsRoomsAndStatus`, then finish with `make test` or `make ci`. If a change touches persistence, TUI state, or helper behavior, add regression coverage in the corresponding `*_test.go` file.

## Commit & Pull Request Guidelines
Recent history favors short imperative commit subjects such as `Fix golangci-lint findings` and `Remember last selected room`. Keep commits scoped to one change. PRs should include a concise summary, validation commands you ran, and screenshots or terminal captures when UI behavior changes. Base feature work on `main` and push reviewable branches instead of mixing unrelated fixes.
Before committing, run the relevant linters and tests for the files or packages you touched, then include that validation in your handoff. Work against `shlomiuziel/sonosh` by default; do not target `steipete/sonoscli` unless explicitly requested.

## Security & Configuration Tips
Do not commit local cache/build artifacts such as `.gocache/`, `.gomodcache/`, `.cache/`, or ad hoc binaries. Prefer repo-local config paths in tests. When handling helper paths, network endpoints, or local files, follow the existing lint-aware patterns already used in the repo.
