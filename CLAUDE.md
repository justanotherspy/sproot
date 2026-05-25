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
pkg/log/              - structured logger (+/-/!/x visual conventions, TTY-aware color)
pkg/table/            - box-drawing table renderer (terminal-fit, ANSI styling)
pkg/tty/              - terminal/color detection shared by log, table consumers
docs/                 - user-facing docs (modules.md)
testdata/integration/ - integration test config used by integration.yml
plans/                - design docs, not shipped
.claude-plugin/       - marketplace.json (this repo is a Claude plugin marketplace)
plugins/sproot/       - the sproot Claude plugin: script-convert + author-config skills, shared reference/
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
| 14 | Intelligence: Claude plugin marketplace with two skills (`script-convert` bash->sproot.yaml converter, `author-config` usage assistant); golden fixtures validated in CI (done) |
| 15-16 | Operational improvements, integration depth, make e2e (done) |
| 17 | Code quality and bug fixes: push env forwarding, HTTP timeouts, label/SHA cleanup, MIGRATION.md; module additions (repo_clone URLs, npm, sprite_service http_port/needs) (done) |
| 18 | Intelligence and completion: llm.txt/agent-context.md after setup (18a done), token scope docs (18b done); config init org auto-select (18c dropped, no SDK org-listing method); release workflow test (18d done, validated by the v0.1.0 release: all 5 archives + checksums + sigstore bundle) |
| 19 | Module gaps to drain cmd blocks: binary_release `version`/`arch_map`/cosign; merge `claude upgrade`+`claude_settings` into a `claude` module (settings/upgrade/CLAUDE.md); sprite-env-aware docker daemon.json merge; apt symlink ~ expansion + mkdir; sprites-artefacts reference snapshot (done) |
| 20 | Self-update: daily cached release check (`~/.sproot/update-check.json`) that notifies after any command; `sproot self-update` (with `--check`) downloads/verifies/replaces the binary and clears the cache; `SPROOT_NO_UPDATE_CHECK` opt-out (done) |
| 21 | Config SHA cache (`~/.sproot/config-cache.json`): `git ls-remote` short-circuits the host-side clone in `sproot new`/`push`/`outdated` when the ref has not moved (17i/17j); new `shell_completion` module (generate + install bash/zsh/fish completions, zsh fpath auto-wire). 19 module types (done) |
| 22 | `nix` module: installs Determinate Nix (`--init none`), runs nix-daemon as a sprite service, declaratively installs profile packages, symlinks the nix CLI + package binaries into `~/.local/bin` (the base PATH `sprite exec`/services inherit), sources the nix profile into the login shells, and runs an optional setup script. 20 module types (done) |
| 23 | `sproot rc` host command: starts/stops Claude Code Remote Control on a sprite as a `claude-rc` sprite service (keeps the sprite awake, persists across reboots) reachable from claude.ai/code + mobile. Writes a launcher script and PUTs the service; flags `--dir`/`--spawn`/`--session-name`/`--close`. Requires a one-time interactive `claude auth login` on the sprite (cached at `/root/.claude/.credentials.json`); forwarded inference tokens are not accepted by Remote Control (done) |

## Workflow

- Each phase or feature goes on its own branch and merges via PR. Never push directly to main.
- Run `make check` before every push (vet + test + lint must all pass).
- When behavior changes, update the relevant docs in the same PR: `docs/modules.md` for module changes, `README.md` for user-facing command or config changes, `CLAUDE.md` phase table for phase completion, and `plans/sproot.md` for design decisions and phase summaries.
- When a module is added, removed, or its fields change, also update the plugin's `plugins/sproot/reference/module-map.md` and `reference/module-schema.md` in the same PR (they back the script-convert/author-config skills), and consider whether a new golden fixture under `plugins/sproot/skills/script-convert/examples/` is warranted.
- Every new CLI command or config feature must have corresponding integration test coverage: unit tests under `internal/host/` or `internal/sprite/`, a dry-run path in `internal/phase/modules/integration_test.go` if a new module type is added, and an entry in `integration.yml` (matrix or separate job) that exercises the feature against a real sprite. When reviewing or implementing a phase, explicitly check that all new functionality is covered end-to-end.
- Sprites spin up in 1-2 seconds. Integration tests do not need artificial sleep or retry loops; commands can run immediately after sprite creation.
- When adding integration tests for config-source functionality, cover BOTH `config_source: git` (standard git clone path) and `config_source: local` (local directory uploaded to sprite). The two code paths are distinct and both must be exercised.

## CI

Three jobs run on every push via `ci.yml`: `build-and-test`, `validate` (runs `sproot validate` against `internal/config/testdata/sproot.yaml` and `sproot_targets.yaml`), and `lint`. All three must pass before merging. golangci-lint uses `.golangci.yml` (standard preset).

`plugins.yml` runs on every push/PR and gates the Claude plugin marketplace: `validate-plugin` installs the `claude` CLI and runs `claude plugin validate --strict` on `.claude-plugin/marketplace.json` and `plugins/sproot`; `validate-fixtures` builds the binary and runs `sproot validate` on every `plugins/sproot/skills/script-convert/examples/*/expected/sproot.yaml` golden fixture.

`integration.yml` runs on owner-triggered pushes: builds the binary and runs matrix integration tests against real sprites. Current jobs include module-type matrix entries (apt, git_identity, file_template, rc_block, claude, cmd, binary-release), tooling matrix entries against `sproot_tooling.yaml` (uv_tool, corepack, rust_components, go_install, cargo_install, npm, sprite_service), a docker-daemon job (docker daemon.json merge + sprite_service dockerd), a multi-target entry (target=web with sproot_targets.yaml), push-and-outdated (creates a sprite, pushes to it, and runs sproot outdated), and local-config (config_source: local using testdata/integration as the local path). `gh_token` and `ssh_setup` are not in the matrix: they need a user-scoped GitHub PAT and mutate the GitHub account, so they cannot run unattended with the default Actions token. `sproot rc` (Remote Control) is likewise not exercised end-to-end: it needs an interactive `claude auth login` (full claude.ai OAuth, not the forwarded inference token), so CI only covers the auth-free paths (the "not authenticated" guard and service register/delete plumbing) via unit tests. The `cargo_install` job uses a small, dependency-light crate (`hexyl`, ~25s to compile on the base image) to keep it fast; avoid swapping in crates with heavy dependency trees.

When a CI job fails, the fastest way to see the errors is the `shuck` CLI (from `justanotherspy/shuck`), which extracts only the failing step logs from a PR's CI runs:

```
shuck                       # failing logs for the current branch's open PR
shuck <pr>                  # PR number, inferred from local repo origin
shuck <owner>/<repo> <pr>   # explicit PR reference
shuck <pr-url>              # from a GitHub PR URL
```

Install with `go install github.com/justanotherspy/shuck@latest` (or the install script in that repo). Useful flags: `--full` (complete logs, not just errors), `--context N` (lines around each error), `--refresh` (rebuild cache).

If `shuck` is unavailable, fall back to the gh CLI:

```
gh run view <run-id> --log-failed
gh run list --branch <branch>   # find the run-id if unknown
```
