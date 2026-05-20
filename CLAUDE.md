# CLAUDE.md

## Project

sproot is a Go CLI that bootstraps sprite.dev sprites from a user-owned config repo. It replaces the current bash-based setup with a generic, reusable tool.

Module path: `github.com/justanotherspy/sproot`
Entry point: `cmd/sproot/main.go`

## Build and test

```
make build       # produces ./sproot binary
make test        # go test ./...
make check       # vet + test + lint (run before pushing)
make tidy        # go mod tidy
make lint        # golangci-lint run ./...
```

## Directory layout

```
cmd/sproot/          - main package and subcommand wiring
internal/config/     - sproot.yaml and ~/.sproot/config schema + loaders
internal/phase/      - Phase interface, runner, state, registry
internal/phase/modules/ - one file per module type (apt, uv_tool, etc.)
internal/host/       - host-side command implementations (new, destroy, status)
internal/sprite/     - in-sprite command implementations (setup)
pkg/log/             - structured logger (+/-/!/x visual conventions)
plans/               - design docs, not shipped
```

## Conventions

- No emdashes anywhere: in code comments, error messages, logs, or docs. Use parens or commas instead.
- Single binary. `sproot` routes by subcommand. No separate host/sprite binaries.
- Files referenced by `sproot.yaml` live in the config repo, not embedded in the binary. The binary embeds only help text, version, and default schemas.
- Idempotency is per-phase, not driven by the state file. The state file is for `--status` and forensics only.
- Log output uses `pkg/log` conventions: `+` success, `-` info, `!` warning, `x` error.
- Version is injected at build time via `-ldflags "-X main.version=vX.Y.Z"`. Default is `dev`.

## Two-repo model

- `justanotherspy/sproot` (this repo): the CLI source and release binaries. Generic, no personal config.
- `justanotherspy/sprite`: the reference config repo. Holds `sproot.yaml` and accompanying files. Fork it or write your own.

## Phase plan

Phases are implemented in order. Each phase has unit tests before the next one starts.

| Phase | Description |
|-------|-------------|
| 0 | Scaffold: module, cobra wiring, CI (done) |
| 1 | Config schema: sproot.yaml and ~/.sproot/config structs + loaders (done) |
| 2 | Phase engine: runner, state file, registry (done) |
| 3 | Module implementations (apt, uv_tool, file_template, etc.) (done) |
| 4 | `sproot setup` (in-sprite command) (done) |
| 5 | Host CLI commands (new, destroy, status, config) (done) |
| 6 | Convert justanotherspy/sprite into a config repo |
| 7 | Release pipeline (goreleaser, sigstore signing) |
| 8 | Docs |

## CI

Two jobs run on every push: `build-and-test` and `lint`. Both must pass before merging.
golangci-lint uses `.golangci.yml` (standard preset).
