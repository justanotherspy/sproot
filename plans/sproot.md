# sproot: plan

A Go CLI for bootstrapping sprite.dev sprites from a user-owned config repo.
Replaces the current bash-based `sprite` repo with a generic, reusable tool.

## Two-repo model

- **`justanotherspy/sproot`** (new): the CLI source + release binaries. Generic, no personal config.
- **`justanotherspy/sprite`** (existing): becomes a reference *config repo*. Holds a `sproot.yaml` plus accompanying files (statusline.py, claude settings, etc). Anyone can fork it or write their own.

## Host file layout

```
~/.sproot/
└── config           # YAML: config_repo, config_ref, token_env, gh_token_env, default_org
```

(The original plan showed a `private/id_ed25519` key here; that model was dropped. Each sprite generates its own keypair via `ssh_setup`.)

## End-to-end flow

```
sproot new my-sprite
  ├─ reads ~/.sproot/config
  ├─ sprites.New(token).CreateSprite(ctx, "my-sprite", nil)
  ├─ sprite.Filesystem().WriteFile("/usr/local/bin/sproot", binaryBytes, 0755)
  ├─ sprite.Command("sproot", "setup", "--config-repo", url).Run()   // streams output
  └─ in-sprite: clone config repo, read sproot.yaml, run phases, verify
```

## Phases (for execution)

Each phase is a discrete deliverable. Phases are roughly in dependency order.

---

### Phase 0: Scaffold (DONE)

- `go.mod` (Go 1.23), cobra, Makefile (build/test/check/lint/tidy)
- Full directory layout, CI (`build-and-test` + `lint`)
- `cmd/sproot/main.go` wired to cobra

---

### Phase 1: Config schema (DONE)

- `internal/config/schema.go`: `SprootConfig`, `HostConfig`, `Identity`, `PhaseConfig` and all typed phase config structs
- `PhaseConfig.UnmarshalYAML`: two-pass custom unmarshaler (reads `type`, decodes into concrete struct using flat form)
- `internal/config/load.go`, `validate.go`
- `internal/config/testdata/sproot.yaml` + `testdata/host_config.yaml`
- 35 unit tests

**`sproot.yaml`** shape (flat; the nested sub-key form shown in `docs/modules.md` is wrong):

```yaml
schema_version: 1
identity:
  git_user_name: "Daniel Schwartz"
  git_user_email: "danielschwar@gmail.com"
  git_default_branch: main
  gh_username: justanotherspy

phases:
  - type: apt
    packages: [shellcheck, jq]
  - type: binary_release
    name: cosign
    repo: sigstore/cosign
    asset: "cosign_{version}_{arch}.deb"
    install: dpkg
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
```

**`~/.sproot/config`** shape:

```yaml
config_repo: git@github.com:justanotherspy/sprite.git
config_ref: main
config_path: ""        # optional; path to config file within repo; defaults to sproot.yaml
token_env: SPRITE_TOKEN
gh_token_env: GITHUB_TOKEN
default_org: ""
```

---

### Phase 2: Phase engine (DONE)

- `internal/phase/phase.go`: `Phase` interface + `Context` struct
- `internal/phase/runner.go`: orchestrates phases, tracks did-work/skipped/failed, `--only`, `--force`
- `internal/phase/state.go`: reads/writes `~/.config/sproot/state.json`
- `internal/phase/registry.go`: `Register` + `Build` with `init()`-based registration
- `pkg/log/log.go`: `+`/`-`/`!`/`x` visual conventions

---

### Phase 3: Module implementations (DONE)

All 17 modules in `internal/phase/modules/`. Each registered via `init()`:
`apt`, `uv_tool`, `go_install`, `cargo_install`, `binary_release`, `corepack`, `rust_components`, `docker`, `sprite_service`, `git_identity`, `ssh_setup`, `gh_token`, `file_template`, `rc_block`, `repo_clone`, `claude_settings`, `cmd`.

- `exec.go`: shared `runCmd` helper
- `modules.go`: blank-import of all module packages
- Unit tests for each module, plus `integration_test.go` (all 17 types under `--dry-run`)
- `docs/modules.md`: module reference (has doc accuracy bugs, see Phase 8)

---

### Phase 4: `sproot setup` (in-sprite command) (DONE)

- `cmd/sproot/setup.go`: cobra command with `--config-repo`, `--ref`, `--config-path`, `--only`, `--force`, `--dry-run`, `--status`
- `internal/sprite/setup.go`: `RunSetup` clones/pulls config repo, loads+validates `sproot.yaml`, runs phases
- `internal/sprite/verify.go`: built-in verify phase (PATH tools, SSH permissions, rc block, gh auth, SSH connectivity)
- `internal/sprite/status.go`: `PrintStatus` renders phase state table

---

### Phase 5: Host CLI commands (DONE)

- `internal/host/client.go`: `SpritesClient` and `SpriteHandle` interfaces wrapping `sprites-go`
- `internal/host/new.go`: `RunNew`
- `internal/host/destroy.go`: `RunDestroy` (reads key IDs from sprite, deletes from GitHub, destroys sprite)
- `internal/host/status.go`: `RunStatus`
- `internal/host/config.go`: `RunConfigInit`, `RunConfigValidate`
- `internal/host/validate.go`: `RunValidate` (validates `sproot.yaml` only)
- `cmd/sproot/`: cobra wiring for all commands

---

### Phase 6: Convert `justanotherspy/sprite` into a config repo

Status: lives in a separate repo, not visible here.

**Deliverables:**
- `sproot.yaml`: full translation of every `setup.sh` phase into module invocations
- `files/`: statusline.py, ps1.sh, rc_additions.sh, gitignore_global, pre-commit-config.template.yaml, claude-settings.json
- `README.md`: explain this is a config example, point at sproot for the tool
- Archive `setup.sh`, `post.sh`, `pre.sh`, `_lib_verify.sh`

**Acceptance:** `sproot new test-sprite` against this repo produces a sprite equivalent to `setup.sh`.

---

### Phase 7: Release pipeline (DONE)

- `.goreleaser.yaml`: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64; tar.gz/zip; checksums + sigstore signing
- `.github/workflows/release.yml`: triggers on `v*` tags
- `install.sh`: one-line installer; detects OS/arch, verifies SHA256, drops binary in `/usr/local/bin` or `~/.local/bin`
- `README.md`: install instructions, quickstart

---

### Phase 8: Doc accuracy fixes and Q1-Q5 code improvements (DONE)

Merged PR #17 on 2026-05-20.

**Q1 (env block): Implemented (option 1).** `RunNew` now clones the config repo before creating the sprite, reads the `env` block from `sproot.yaml`, resolves each `from` var via `os.Getenv`, fails hard if `required: true` and the var is unset, and appends resolved vars to the env slice forwarded to `sproot setup`. The existing `gh_token_env` forwarding is preserved as a baseline.

**Q2 (file_template opt-in): Added `Template bool` (option 2).** Go template rendering is now opt-in via `template: true` in the phase config. Without it the file is copied as-is. Prevents silent substitution of unintended `{{...}}` patterns.

**Q3 (cmd name field): Added (option 1).** `CmdConfig` gains `name: string`. `Name()` returns `cmd(foo)` when set, making multiple `cmd` phases distinguishable in status output.

**Q4 (binary_release checksums): Both options shipped.** `BinaryReleaseConfig` gains `checksum` (direct sha256 hex) and `checksum_asset` (goreleaser-style checksums file template, e.g. `{repo}_{version}_checksums.txt`). Both are optional; either is verified after download and before install.

**Q5 (validate commands): Partially combined.** `sproot validate` now also validates `~/.sproot/config` when the file exists, in addition to `sproot.yaml`. The README commands table was corrected to distinguish `config validate` (host config only) from `validate` (sproot.yaml + host config if present).

Doc fixes shipped (8a-8j):
- All 15 nested YAML examples in `docs/modules.md` rewritten to flat form
- `admin:public_key`/`admin:ssh_signing_key` scopes moved from `gh_token` to `ssh_setup` section
- `GH_TOKEN` env reference corrected in `gh_token` and `ssh_setup` sections
- `gh auth login` example updated with `--hostname github.com`
- README commands table fixed, `config_path` added to host config example
- `CLAUDE.md` phase table, CI section, and directory layout updated

---

### Phase 9: Bug fixes (DONE)

Merged PR #18 on 2026-05-20. All six items fixed with unit tests added for each.

**9a. `rc_block` `ShouldRun` checks both shells.** `ShouldRun` now iterates `.bashrc` and `.zshrc` and returns true if either is missing or has a stale hash, matching the two-file write in `Run`. Previously only `.bashrc` was checked, causing `Verify` to fail on `.zshrc` after `ShouldRun` returned false.

**9b. `rc_block` trailing newline normalised.** `applyRCBlock` ensures `src` ends with `\n` before composing the block, so the end sentinel always appears on its own line regardless of source file content.

**9c. `binary_release` GitHub API auth.** `githubLatestTag` now sends `Authorization: Bearer $GH_TOKEN` when the var is set, avoiding the 60 req/hr anonymous rate limit on shared runners.

**9d. `binary_release` HTTP timeouts.** Two module-level `*http.Client` values replace bare `http.Get` calls: `tagClient` (30s) for API and checksums requests, `downloadClient` (5m) for asset downloads.

**9e. `ssh_setup` idempotency gap closed.** `ShouldRun` now also returns true when `~/.ssh/allowed_signers` does not contain the local pubkey, or when `GH_TOKEN` is set but `~/.config/sproot/github_keys.json` is absent (key was generated but GitHub registration was skipped on a prior run without the token).

**9f. `cloneOrPull` handles changed remote URL.** Before fetching, `cloneOrPull` compares `git remote get-url origin` against the requested URL. If they differ it runs `git remote set-url origin <new>` so a changed `config_repo` takes effect without requiring `--force` or a manual re-clone.

---

### Phase 10: Binary injection cross-arch fix (DONE)

Merged PR #21 on 2026-05-20.

`RunNew` now auto-detects the host platform at runtime:

- **Linux/amd64**: reads and injects its own executable (unchanged from before).
- **Any other platform** (macOS arm64, etc.): downloads the matching Linux/amd64 release tarball from GitHub using the running binary's version, extracts the `sproot` binary, and injects that.

Key details:
- `internal/host/fetch.go`: `fetchLinuxAmd64Binary(version)` and `extractSprootFromTarGz(r)`.
  - Returns a clear error when `version == "dev"` (no release to download).
  - Handles binary at tarball top level or inside a goreleaser-style prefixed directory.
- `NewOptions.binarySrcFn` injection point (`func(version string) ([]byte, error)`) lets tests override binary sourcing without platform detection.
- `NewOptions.Version` carries the build version from `main.version` (set by goreleaser ldflags).
- Q6 resolved: option 2 (download at runtime).

---

### Phase 11: UX improvements (DONE)

Merged PR #22 on 2026-05-21.

**11a (interactive config init): Done.** `sproot config init` now prompts for `config_repo`, `token_env`, `gh_token_env`, `default_org`; validates; writes the file. `--non-interactive` flag preserves the old skeleton-file behavior for scripting.

**11b (token scope docs): Deferred.** Low-priority doc-only change; can be added alongside Phase 12 SDK docs.

**11c (`sproot new` opens console): Done.** After `sproot setup` completes successfully, `RunNew` opens an interactive TTY shell. `--no-console` and `--dry-run` both skip the console step.

**11d (`sproot console <name>`): Done.** `internal/host/console.go` + `cmd/sproot/console.go`. Opens `bash` with TTY on the named sprite. Terminal size is read from the host; raw mode is enabled when stdin is a TTY.

**11e (`sproot list`): Done.** `internal/host/list.go` + `cmd/sproot/list.go`. Sprites created by `sproot new` are tagged with the `"sproot"` label via `CreateSpriteWithOrg`. `sproot list` filters to that label; `--all` shows every sprite.

**11f (auto-setup config): Done.** `loadOrInitHostConfig` in `config.go` detects a missing `~/.sproot/config`, prints a warning, and offers the interactive init flow when stdin is a terminal. Used by `RunNew`, `RunStatus`, `RunDestroy`, `RunConsole`, `RunList`.

**11g (debug logging): Done.** `--debug` global flag on the root cobra command calls `log.SetDebug(true)`. `Logger.Debug/Debugf` write `. <msg>` lines only when debug is enabled.

**11h (validate sproot.yaml before creating sprite): Done.** `readEnvBlock` now calls `config.ValidateSprootConfig` after loading, so a broken `sproot.yaml` fails before any sprite API quota is spent.

**11i (validate reachability): Deferred.** Would require cloning the config repo a second time in `sproot validate`; not worth the cost for a warning. Revisit if users report confusion.

---

### Phase 12: Sprite wrapping and SDK alignment (DONE)

Merged PR #23 on 2026-05-21.

**12a: Additional host command wrappers (done).**
- `sproot exec <name> <cmd> [args...]`: runs a one-off command in a sprite and streams stdout/stderr.
- `sproot upgrade <name>`: upgrades a sprite to the latest version via SDK `UpgradeSprite`.
- `sproot checkpoint <name> [--comment text]`: creates a checkpoint, streams progress to stdout.
- `sproot checkpoints <name> [--include-auto]`: lists checkpoints in a table.
- `sproot restore <name> <id>`: restores a sprite from a checkpoint, streams progress to stdout.

**12b: SDK audit (done).**
- Audited all sprites-go SDK methods against sproot usage. All calls map to real SDK methods.
- Added `--storage-gb` to `sproot new` (maps to `SpriteConfig.StorageGB`).
- Added `Upgrade`, `Checkpoint`, `ListCheckpoints`, `Restore` to `SpriteHandle` interface and `realHandle`.
- Added `UpgradeSprite` to `SpritesClient` interface and `realClient`.

**12c: Checkpointing integration (done).**
- `SprootConfig.CheckpointAfterSetup bool` added to `internal/config/schema.go`.
- `RunNew` calls `handle.Checkpoint(ctx, "sproot setup", ...)` after successful setup when `checkpoint_after_setup: true` in `sproot.yaml`. Checkpoint failure is a warning, not a fatal error.

**12d: Host-readable phase state (done).**
- `sproot status <name> --host` reads `/root/.config/sproot/state.json` from the sprite filesystem via `ReadFile` and renders the phase table locally without exec'ing into the sprite.
- Default (no `--host`) retains the original behavior: runs `sproot setup --status` inside the sprite.

---

### Phase 13: Multi-target and push

#### 13a. Multiple sproot targets in one `sproot.yaml`

Extend `sproot.yaml` to support named targets:
```yaml
targets:
  base:
    phases: [...]
  web:
    extends: base
    phases: [...]
```
`sproot new my-sprite --target web` runs only the `web` target's phases. The `extends` field inherits the parent's phase list. Without a target flag, a default target (or the flat `phases:` block) is used. This enables one config repo to produce several specialized sprite flavors.

**OPEN QUESTION**: flat `phases:` backward compat -- should a `sproot.yaml` with only `phases:` still work unchanged, or require migration? Recommendation: yes, treat flat `phases:` as an implicit `default` target.

#### 13b. Local path as config source

Currently `sproot.yaml` must live in a git repo. Add support for a local path:

```yaml
# ~/.sproot/config
config_source: local     # "git" (default) or "local"
config_local_path: ~/my-sprite-config/sproot.yaml
```

Or via flag: `sproot new --config-path /path/to/sproot.yaml` without a `config_repo`. Useful for development and for passing a config directly from the host without a git remote.

#### 13c. `sproot push` / `sproot update`

Push a config change to all sprites created by sproot (identified by the sproot label from Phase 11e):
- `sproot push`: pull latest from `config_repo` for each sprite and re-run setup (wake sprites, run `sproot setup --force`, optionally checkpoint before updating).
- `sproot push --target <name>`: push to a specific sprite by name.
- Run pushes in parallel with progress output.

**OPEN QUESTION**: should `sproot push` checkpoint before updating? Recommendation: yes, always checkpoint before a push so the user can restore if the update breaks something.

---

### Phase 14: Intelligence and agent context

#### 14a. Update llm.txt and agent-context.md after setup

After `sproot setup` completes, append a summary of what was installed to `/.sprite/llm.txt` and `/.sprite/docs/agent-context.md` inside the sprite. Each phase that ran should add a line: tool name, version if known, and why it is useful. This gives Claude Code instant context about the environment it is operating in.

#### 14b. Claude skill for sproot usage

Once docs and UX are stable, create an installable skill that enables Claude Code to:
- Generate a `sproot.yaml` from a description or from an existing setup script.
- Validate and explain phase configurations.
- Suggest which modules to use for a given requirement.

#### 14c. Convert a script into a sproot.yaml (skill)

A skill that takes an existing setup script (bash or other) as input and generates the equivalent `sproot.yaml`. Map common patterns: `apt-get install` -> `apt`, `pip install` -> `uv_tool`, curl-pipe-sh installs -> `binary_release` or `cmd`, git config -> `git_identity`, etc.

---

## Open questions summary

| # | Area | Question | Blocks | Status |
|---|------|----------|--------|--------|
| Q1 | `env` block | Implement, drop, or mark as future? | 8i, arch of env forwarding | DONE: implemented (option 1) in PR #17 |
| Q2 | `file_template` | Add `Template bool` opt-in or document always-attempt? | 8a | DONE: `Template bool` added (option 2) in PR #17 |
| Q3 | `cmd` `name` field | Add field or drop from docs? | 8a | DONE: field added (option 1) in PR #17 |
| Q4 | `binary_release` checksums | Add optional `checksum:` field or known tradeoff? | 9d | DONE: both `checksum` and `checksum_asset` added in PR #17 |
| Q5 | validate commands | Keep separate or combine? | 8b | DONE: partially combined; `sproot validate` now also validates host config if present, in PR #17 |
| Q6 | Cross-arch binary injection | Embed Linux/amd64 binary, download it at runtime, or document limitation? | Phase 10 | DONE: download at runtime (option 2) in PR #21 |
| Q7 | Multi-target backward compat | Flat `phases:` in Phase 13a treated as implicit default target? | Phase 13a | OPEN |

---

## Suggested order of execution

1. **Phase 9 bug fixes** (no questions blocking): 9a, 9b, 9c, 9d, 9e, 9f.
2. **Phase 10** (cross-arch injection fix): depends on answer to Q6; affects all non-Linux users.
3. **Phase 11 UX** (11a through 11i): independent; can be picked off one at a time.
4. **Phase 12 SDK alignment** (12a-12d): review SDK docs first.
5. **Phase 13 multi-target and push** (13a-13c): architecture changes; coordinate with Phase 6 (sprite config repo).
6. **Phase 14 intelligence** (14a-14c): after everything else is stable.

After each batch of changes: `make check` and `./sproot validate --path internal/config/testdata/sproot.yaml`.

---

## Cross-cutting constraints

- **No emdashes anywhere** in code comments, error messages, logs, or docs. Use parens or commas.
- **Single binary.** `sproot` routes by subcommand. No separate host/sprite binaries.
- **Embedding strategy.** Files referenced in `sproot.yaml` live in the config repo, not embedded in the sproot binary. The binary embeds only its own help text, version, and default schemas.
- **Host-sprite interaction uses sprites-go SDK.** Never shell out to the `sprite` CLI from Go code.
- **Bracket phases (pre/post sprite-env checkpoints) are a CLI concern, not a module type.** They wrap the whole setup run and never abort it.
- **Idempotency is per-phase, not driven by the state file.** State file is for `--status` and forensics.
- **Feature branches for PRs.** Each body of work goes on a feature branch and merges via PR. Never push directly to main.
