# sproot: plan

A Go CLI for bootstrapping sprite.dev sprites from a user-owned config repo.
Replaces the current bash-based `sprite` repo with a generic, reusable tool.

## Two-repo model

- **`justanotherspy/sproot`** (new): the CLI source + release binaries. Generic, no personal config.
- **`justanotherspy/sprite`** (existing): becomes a reference *config repo*. Holds a `sproot.yaml` plus accompanying files (statusline.py, claude settings, etc). Anyone can fork it or write their own.

## Host file layout

```
~/.sproot/
├── config           # YAML: token, config_repo_url, private_key_path, defaults
└── private/
    └── id_ed25519   # user's sprite SSH key (loaded from password manager)
```

## End-to-end flow

```
sproot new my-sprite
  ├─ reads ~/.sproot/config
  ├─ sprites.New(token).CreateSprite(ctx, "my-sprite", nil)
  ├─ sprite.Filesystem().WriteFile(".ssh/id_ed25519", keyBytes, 0600)
  ├─ sprite.Filesystem().WriteFile("/usr/local/bin/sproot", binaryBytes, 0755)
  ├─ sprite.Command("sproot", "setup", "--config-repo", url).Run()   // streams output
  └─ in-sprite: clone config repo, read sproot.yaml, run phases, verify
```

## Phases (for execution)

Each phase is a discrete deliverable. Phases are roughly in dependency order. Tests are required inline per phase, not deferred.

---

### Phase 0: Scaffold `justanotherspy/sproot` (DONE)

**What was built:**
- `go.mod` (Go 1.23), cobra for command routing, Makefile with build/test/check/lint/tidy targets
- Full directory layout: `cmd/sproot/`, `internal/config/`, `internal/phase/`, `internal/phase/modules/`, `internal/host/`, `internal/sprite/`, `pkg/log/`
- `.github/workflows/ci.yml`: build-and-test + lint jobs on every push
- `cmd/sproot/main.go` wired to cobra; `internal/host/` and `internal/sprite/` are empty stubs

---

### Phase 1: Config schema (DONE)

**What was built:**
- `internal/config/schema.go`: `SprootConfig`, `HostConfig`, `Identity`, `PhaseConfig` and all typed phase config structs with `yaml:""` tags. `PhaseConfig` uses a two-pass custom unmarshaler: first reads `type`, then decodes into the matching concrete struct pointer.
- `internal/config/load.go`: `LoadSprootConfig` and `LoadHostConfig` with `~` expansion
- `internal/config/validate.go`: `ValidateSprootConfig` and `ValidateHostConfig` with field-named error messages; detects unknown module types
- `internal/config/testdata/sproot.yaml` + `testdata/host_config.yaml`
- 35 unit tests covering happy path and every validation failure

**`sproot.yaml`** shape:
```yaml
schema_version: 1

identity:
  git_user_name: "Daniel Schwartz"
  git_user_email: "danielschwar@gmail.com"
  git_default_branch: main
  gh_username: justanotherspy

phases:
  - type: apt
    packages: [shellcheck, jq, ...]
  - type: uv_tool
    tools: [{name: ruff}, {name: pre-commit}]
  - type: binary_release
    name: cosign
    repo: sigstore/cosign
    asset: "cosign_{version}_{arch}.deb"
    install: dpkg
  - type: go_install
    tools: [{pkg: "github.com/owner/tool", version: latest}]
  - type: cargo_install
    tools: [{name: ripgrep}]
  - type: file_template
    src: files/statusline.py
    dest: ~/.claude/statusline.py
    mode: "0755"
  - type: rc_block
    src: files/rc_additions.sh
  - type: repo_clone
    base_dir: ~/repos
    repos:
      - justanotherspy/garlic
      - justanotherspy/poker
```

**`~/.sproot/config`** shape:
```yaml
config_repo: git@github.com:justanotherspy/sprite.git
config_ref: main
token_env: FLY_API_TOKEN    # env var name holding the Fly/sprites API token
gh_token_env: GITHUB_TOKEN  # env var name holding the GitHub PAT
default_org: ""
```

The config stores environment variable **names**, not token values. `sproot` calls `os.Getenv(name)` at runtime. This keeps secrets out of the config file and lets each machine use whatever token source it prefers (password manager export, `.profile`, CI secret, etc.).

---

### Phase 2: Phase engine (DONE)

**What was built:**
- `internal/phase/phase.go`: `Phase` interface (`Type`, `Name`, `ShouldRun`, `Run`, `Verify`) and `Context` struct (config repo path, identity, logger, dry-run, force flags)
- `internal/phase/runner.go`: orchestrates the phase list; tracks did-work / skipped / failed buckets; supports `--only` (by name or index) and `--force`
- `internal/phase/state.go`: reads/writes `~/.config/sproot/state.json`; records per-phase run time, status, and error
- `internal/phase/registry.go`: `Register` + `Build` with `init()`-based module registration
- `pkg/log/log.go`: `+`/`-`/`!`/`x` visual conventions with color support
- Full test coverage: runner tests with dummy phases (pass, fail, skip, force), state round-trip tests, registry tests

---

### Phase 3: Phase module implementations (DONE)

**What was built:**
- All 17 module files in `internal/phase/modules/`, each registered via `init()`:
  - `apt` — install apt packages
  - `uv_tool` — install via `uv tool install`
  - `go_install` — `go install pkg@version`; idempotency via `go version -m $(which <binary>)`
  - `cargo_install` — `cargo install`; supports optional `version`, `features`, `locked`; idempotency via `cargo install --list`
  - `binary_release` — GitHub releases with asset templating (`{version}`, `{arch}`, `{goos}`, `{dpkg_arch}`) and install methods (`dpkg`, `tar+install`, `raw`)
  - `corepack` — enable + pre-activate pnpm and yarn
  - `rust_components` — pin stable, install clippy/rustfmt/rust-analyzer via `rustup`
  - `docker` — docker-ce install + `/etc/docker/daemon.json`
  - `sprite_service` — register a `sprite-env` service (dockerd today, extensible)
  - `git_identity` — user.name, user.email, default branch, aliases, signing config
  - `ssh_setup` — chmod key, populate known_hosts, derive pubkey via `ssh-keygen -y -f`, write `allowed_signers`
  - `gh_token` — export `GH_TOKEN` from sprite app token; verify `gh auth status`
  - `file_template` — copy file from config repo to dest, with optional Go-template execution against identity
  - `rc_block` — write sentinel-delimited block to `.bashrc` and `.zshrc`; strips legacy blocks
  - `repo_clone` — clone a list of `git@github.com:owner/repo` repos into a base dir
  - `claude_settings` — deep-merge a JSON object into `~/.claude/settings.json`
  - `cmd` — escape hatch: arbitrary command with optional idempotency-check command
- `exec.go` — shared `runCmd` helper that streams stdout+stderr line-by-line via `pkg/log`
- `modules.go` — blank-import of all module packages to trigger `init()` registration
- Unit tests for each module, plus `integration_test.go` covering all 17 types under `--dry-run`
- `docs/modules.md` — full module reference with YAML examples

---

### Phase 4: `sproot setup` (in-sprite command) (DONE)

**What was built:**
- `cmd/sproot/setup.go`: cobra command with flags `--config-repo` (required), `--ref`, `--only`, `--force`, `--dry-run`, `--status`
- `internal/sprite/setup.go`: `RunSetup` clones or pulls the config repo, loads and validates `sproot.yaml`, builds phase objects, appends the built-in verify phase, and runs the phase runner
- `internal/sprite/verify.go`: built-in verify phase checks PATH commands (git, gh, go, cargo, uv, docker, node, pnpm, rustup), SSH key permissions, RC block presence, `gh auth status`, and SSH connectivity to github.com
- `internal/sprite/status.go`: `PrintStatus` loads the state file and renders a formatted table (type, name, status, ran-at, error)
- Also updated `ssh_setup` module: generates a fresh ed25519 keypair inside the sprite (not injected from host), registers it with GitHub using `GH_TOKEN`, and writes the auth/signing key IDs to `~/.config/sproot/github_keys.json` for cleanup on destroy

---

### Phase 5: Host CLI commands (DONE)

**Auth design (differs from original plan):**
- No private key on the host. Each sprite generates its own keypair via `ssh_setup`.
- `HostConfig` stores env var **names** (`token_env`, `gh_token_env`), not token values. `sproot` calls `os.Getenv(name)` at runtime to resolve them.
- `sproot new` forwards `GH_TOKEN` into the sprite via `cmd.Env` so `ssh_setup` can register the generated key with GitHub.
- `sproot destroy` reads the key IDs from `/root/.config/sproot/github_keys.json` in the sprite (written by `ssh_setup`), deletes them from GitHub using the host's GH token, then destroys the sprite.

**What was built:**
- `internal/config/schema.go`: `HostConfig` replaced `PrivateKey` with `TokenEnv` and `GHTokenEnv`
- `internal/host/client.go`: `SpritesClient` and `SpriteHandle` interfaces wrapping `sprites-go`; `NewClient(token)` for production use; interfaces enable unit testing without real HTTP calls
- `internal/host/new.go`: `RunNew` creates sprite, injects sproot binary via `sprite.Filesystem().WriteFile`, runs `sproot setup` with `GH_TOKEN` in env
- `internal/host/destroy.go`: `RunDestroy` reads GitHub key IDs from sprite filesystem, deletes them via GitHub API (warnings on failure), then calls `client.DestroySprite`
- `internal/host/status.go`: `RunStatus` streams `sproot setup --status` from the sprite
- `internal/host/config.go`: `RunConfigInit` writes skeleton config; `RunConfigValidate` loads and validates
- `cmd/sproot/new.go`, `destroy.go`, `sprite_status.go`, `config.go`: cobra wiring for all four commands
- `go.mod`: added `github.com/superfly/sprites-go` dependency
- Full unit test coverage for all host commands; mock `SpritesClient`/`SpriteHandle` in tests

**Acceptance:** `sproot new my-sprite` produces a working sprite end-to-end against `justanotherspy/sprite` as the config repo; `sproot destroy my-sprite` cleans up GitHub SSH keys and removes the sprite.

---

### Phase 6: Convert `justanotherspy/sprite` into a config repo

Migrate the existing bash repo into a reference sproot config.

**Deliverables:**
- `sproot.yaml`: full translation of every current `setup.sh` phase into module invocations
- `files/`:
  - `statusline.py` (already exists; move here)
  - `ps1.sh` (extract from current `phase_ps1`)
  - `rc_additions.sh` (extract from current `RC_BLOCK`)
  - `gitignore_global` (extract from current `phase_gitignore_global`)
  - `pre-commit-config.template.yaml` (extract from current `phase_pre_commit_template`)
  - `claude-settings.json` (the managed-keys subset)
- `README.md`: explain that the repo is now a config example, point at `justanotherspy/sproot` for the tool
- Archive or delete `setup.sh`, `post.sh`, `pre.sh`, `_lib_verify.sh`. Keep `CLAUDE.md` (refresh contents).

**Acceptance:** `sproot new test-sprite` against this repo produces a sprite functionally identical to what `setup.sh` produces today. Any intentional differences listed in the migration notes.

---

### Phase 7: Release pipeline

Cut versioned binaries.

**Deliverables:**
- `.goreleaser.yaml`:
  - Targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
  - Archives: tar.gz for unix, zip for windows
  - Checksums + sigstore signing
- `.github/workflows/release.yml`: triggers on `v*` tags
- `install.sh`: one-line installer for unix hosts. Detects OS/arch, downloads the right tarball, drops the binary in `/usr/local/bin` (root) or `~/.local/bin` (user)
- `README.md` on the sproot repo: install instructions, quickstart

**Acceptance:** `git tag v0.1.0 && git push --tags` produces a GitHub release with all binaries, checksums, and signatures. `curl -fsSL https://raw.githubusercontent.com/justanotherspy/sproot/main/install.sh | sh` installs a working binary on Linux and macOS.

---

### Phase 8: Docs

Final user-facing docs across both repos.

**Deliverables:**
- `sproot/README.md`: install, quickstart (`sproot config init`, set values, `sproot new my-sprite`)
- `sproot/docs/modules.md`: every module type with YAML example and notes
- `sproot/docs/config.md`: full `sproot.yaml` reference
- `sproot/docs/host-config.md`: `~/.sproot/config` reference
- `sproot/docs/auth-model.md`: the host-key + sprite-app-token split, why, security tradeoffs
- `sprite/README.md`: "this is a sproot config repo. fork it or write your own."

**Acceptance:** all internal doc links resolve. A new user can go from zero to a working sprite by following only the README.

---

## Cross-cutting constraints

- **No emdashes anywhere** in code comments, error messages, logs, or docs. Use parens or commas.
- **Single binary.** `sproot` routes by subcommand. No separate host/sprite binaries.
- **Embedding strategy.** Files referenced in `sproot.yaml` live in the config repo, not embedded in the sproot binary. The binary embeds only its own help text, version, and default schemas.
- **Host sprite interaction uses sprites-go SDK.** Never shell out to the `sprite` CLI from Go code. Use `github.com/superfly/sprites-go` for all sprite lifecycle and exec operations.
- **Bracket phases (pre/post sprite-env checkpoints) are a CLI concern, not a module type.** They wrap the whole setup run and never abort it.
- **The current `pre.sh` is reference only.** Don't port it; the built-in verify phase covers the same ground.
- **Idempotency is per-phase, not driven by the state file.** State file is for `--status` and forensics.
