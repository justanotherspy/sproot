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

## Local development environment

The `sprite` CLI is required to run integration tests and host commands against real sprites. In the cloud execution environment, `SPRITES_TOKEN` is set automatically. Authenticate once per session:

```
sprite auth setup --token "$SPRITES_TOKEN"
```

## Directory layout

```
cmd/sproot/           - main package and subcommand wiring
internal/config/      - sproot.yaml and ~/.sproot/config.yaml schema + loaders
internal/phase/       - Phase interface, runner, state, registry
internal/phase/modules/ - one file per module type (apt, uv_tool, etc.)
internal/host/        - host-side command implementations (new, destroy, status)
internal/sprite/      - in-sprite command implementations (setup)
pkg/log/              - structured logger (+/-/!/x visual conventions)
docs/                 - user-facing docs (modules.md)
testdata/integration/ - integration test config used by integration.yml
plans/                - design docs, not shipped
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

| Phase | Description |
|-------|-------------|
| 0-7 | Foundation: scaffold, config schema, phase engine, 17 modules, sproot setup, host CLI, sprite config repo, release pipeline (done) |
| 8-13 | Hardening: doc fixes, bug fixes, cross-arch binary, UX, SDK alignment, multi-target/push (done) |
| 14 | Intelligence: Claude skills for sproot usage and script conversion (deferred) |
| 15-16 | Operational improvements, integration depth, make e2e (done) |
| 17 | Code quality and bug fixes: push env forwarding, HTTP timeouts, label/SHA cleanup, MIGRATION.md; module additions (repo_clone URLs, npm, sprite_service http_port/needs) (done) |
| 18 | Intelligence and completion: llm.txt/agent-context.md after setup (18a done), token scope docs (18b done); config init org auto-select (18c dropped, no SDK org-listing method); release workflow test (18d deferred) |

## Workflow

- Each phase or feature goes on its own branch and merges via PR. Never push directly to main.
- Run `make check` before every push (vet + test + lint must all pass).
- When behavior changes, update the relevant docs in the same PR: `docs/modules.md` for module changes, `README.md` for user-facing command or config changes, `CLAUDE.md` phase table for phase completion, and `plans/sproot.md` for design decisions and phase summaries.
- Every new CLI command or config feature must have corresponding integration test coverage: unit tests under `internal/host/` or `internal/sprite/`, a dry-run path in `internal/phase/modules/integration_test.go` if a new module type is added, and an entry in `integration.yml` (matrix or separate job) that exercises the feature against a real sprite. When reviewing or implementing a phase, explicitly check that all new functionality is covered end-to-end.
- Sprites spin up in 1-2 seconds. Integration tests do not need artificial sleep or retry loops; commands can run immediately after sprite creation.
- When adding integration tests for config-source functionality, cover BOTH `config_source: git` (standard git clone path) and `config_source: local` (local directory uploaded to sprite). The two code paths are distinct and both must be exercised.

## CI

Three jobs run on every push via `ci.yml`: `build-and-test`, `validate` (runs `sproot validate` against `internal/config/testdata/sproot.yaml` and `sproot_targets.yaml`), and `lint`. All three must pass before merging. golangci-lint uses `.golangci.yml` (standard preset).

`integration.yml` runs on owner-triggered pushes: builds the binary and runs matrix integration tests against real sprites. Current jobs: six module-type matrix entries (apt, git_identity, file_template, rc_block, claude_settings, cmd), a multi-target entry (target=web with sproot_targets.yaml), push-and-outdated (creates a sprite, pushes to it, and runs sproot outdated), and local-config (config_source: local using testdata/integration as the local path).

When a CI job fails, always fetch the full logs using the gh CLI before diagnosing:

```
gh run view <run-id> --log-failed
gh run list --branch <branch>   # find the run-id if unknown
```
