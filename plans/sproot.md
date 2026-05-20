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

### Phase 8: Doc accuracy fixes (PARTIALLY DONE, bugs exist)

Docs exist but contain errors found in the external review. Items below are confirmed fixes (no open questions blocking them).

#### 8a. `docs/modules.md`: all 17 YAML examples use wrong nested form (CRITICAL)

Every example shows `- type: apt\n  apt:\n    packages:` but the unmarshaler decodes the flat node directly. The nested sub-key has no matching field; values silently stay at zero, causing validation errors like `src is required`. Fix: rewrite all 17 YAML blocks to use flat form as in `internal/config/testdata/sproot.yaml`.

#### 8b. `README.md` commands table: `sproot config validate` description is wrong

Current: "Validate host config and sproot.yaml". Correct: only validates `~/.sproot/config`. Add a separate row for `sproot validate [--path PATH]` which validates `sproot.yaml`.

#### 8c. `README.md` "How it works" step 2 is misleading

Current: "Resolves tokens from your environment (`FLY_API_TOKEN`, `GITHUB_TOKEN`, etc.)" -- these names are not hardcoded. Reword to: "Resolves the API and GitHub tokens from the env vars named in `~/.sproot/config` (`token_env` and `gh_token_env`)."

#### 8d. `README.md` host config example missing `config_path`

`HostConfig` has `ConfigPath` but the README example omits it. Add as a commented optional field. Also add to `configSkeleton` in `internal/host/config.go`.

#### 8e. `CLAUDE.md`: phase table is stale

Phase 7 and Phase 8 are shown as not done but are shipped (partially). Update: mark 7 done, mark 8 partial. Add note: "Phase 8 doc accuracy tracked in plans/findings.md."

#### 8f. `CLAUDE.md`: CI section undercounts jobs

"Two jobs run on every push" is wrong. `ci.yml` has three: `build-and-test`, `validate`, `lint`. `integration.yml` adds a build job and six matrix integration tests (owner only). Update both.

#### 8g. `CLAUDE.md`: directory layout missing `docs/` and `testdata/integration/`

Add:
```
docs/                - user-facing docs (modules.md, etc.)
testdata/integration/ - integration test config used by integration.yml
```

#### 8h. `docs/modules.md` `gh_token`: required scopes listed under wrong module

`admin:public_key` and `admin:ssh_signing_key` are needed by `ssh_setup`, not `gh_token`. Move or duplicate to the `ssh_setup` section.

#### 8i. `docs/modules.md` `gh_token` and `ssh_setup`: env block reference is wrong

Both sections say `GH_TOKEN` is "injected via the env block". It is not: it arrives via `gh_token_env` in `~/.sproot/config` forwarded by `RunNew`. Fix both references.

#### 8j. `docs/modules.md` `gh_token`: flags in example differ from code (nit)

Docs: `gh auth login --with-token --git-protocol ssh`. Code: `gh auth login --hostname github.com --git-protocol ssh --with-token`. Add `--hostname github.com` to the docs.

---

### Phase 8 open questions (must answer before acting on these items)

**Q1: `env` block (finding 3)**

Schema, parsing, and validation exist for `env:` in `sproot.yaml` but nothing reads the parsed values at runtime. `GH_TOKEN` arrives via `gh_token_env` in `~/.sproot/config`.

Options:
1. **Implement it**: `RunNew` reads `sproot.yaml` from the config repo before creating the sprite, resolves each `from` env var via `os.Getenv`, fails if `required: true` and unset, builds the env slice passed to `sproot setup`. The `env` block then replaces the `gh_token_env` forwarding logic over time.
2. **Drop it**: remove `EnvVar`, the `Env` field on `SprootConfig`, the validation cases, the testdata entries, and the docs section.
3. **Mark as future**: leave parsing in place, add a TODO comment in `schema.go`, note in `docs/modules.md` that the block parses but does nothing yet.

Recommendation: option 1 is the most flexible and makes the architecture cleaner (generic env forwarding vs. a single hardcoded `GH_TOKEN` field). Option 2 is safe if the single-token model is sufficient indefinitely. Option 3 is confusing in the interim.

**Q2: `file_template` `template:` flag (finding 2)**

`FileTemplateConfig` has no `Template` field. `render()` always attempts `template.Parse` and falls back silently to literal on parse error. This means a file with `{{ .GitUserName }}` (unintentional) will be substituted.

Options:
1. **Document the actual behavior**: always-attempt with fallback; no flag. Lower effort.
2. **Add `Template bool`**: gate template parsing on the field; literal by default. Safer.

Recommendation: option 2 prevents silent surprises. Add `Template bool \`yaml:"template"\`` to `FileTemplateConfig`, gate parse on it.

**Q3: `cmd` module `name` field (finding 4)**

Docs show a `name:` field but `CmdConfig` has none. `cmdPhase.Name()` always returns `"cmd"`. Multiple `cmd` phases in one `sproot.yaml` are indistinguishable in state output.

Options:
1. **Add `Name string \`yaml:"name"\``**: `cmdPhase.Name()` returns `fmt.Sprintf("cmd(%s)", p.cfg.Name)` when set, else `"cmd"`. Matches the pattern of `binary_release(cosign)` and `file_template(<dest>)`.
2. **Drop from docs**.

Recommendation: option 1 is unambiguously useful when using multiple `cmd` phases.

**Q4: `binary_release` checksum verification (finding 7c)**

`downloadAsset` pulls artifacts with no verification. sproot itself uses cosign for releases.

Options:
1. **Add optional `checksum:` field** (sha256 hex string) to `BinaryReleaseConfig`. Verify before installing. Update schema, template logic, run flow, docs.
2. **Add optional `checksum_asset:` field** (template like `{repo}_{version}_checksums.txt`), download the checksums file, parse and verify.
3. **Known tradeoff**: leave as-is, noting the gap in docs.

Recommendation: option 1 is the pragmatic fix for high-value tools. Option 2 handles the common goreleaser/cosign pattern where a checksums file is published.

**Q5: Combining validate commands (finding 5a)**

`sproot config validate` validates only `~/.sproot/config`. `sproot validate` validates `sproot.yaml` only. The README conflates them.

Options:
1. **Keep separate**: fix the README description only (already captured in 8b).
2. **Combine**: `sproot validate [--config PATH] [--sproot-yaml PATH]` checks both files when both flags present.

Recommendation: keep separate. They validate different things used at different times.

---

### Phase 9: Bug fixes

These are real behavior bugs found in the external review. Can land independently.

#### 9a. `rc_block` `ShouldRun` only checks `.bashrc` (BUG)

`ShouldRun` reads only `~/.bashrc` for the block hash. `Run` writes both `.bashrc` and `.zshrc`. If `.bashrc` is current but `.zshrc` is missing or stale, `ShouldRun` returns false, `Run` is skipped, then `Verify` fails on `.zshrc`.

Fix: check both `.bashrc` and `.zshrc` in `ShouldRun`. Return true if either is missing or has the wrong hash. Add a test for the case where only one file is stale.

#### 9b. `rc_block` trailing newline not guaranteed

`applyRCBlock` formats the block as `"\n%s\n%s%s\n"`. If `src` does not end with `\n`, the end sentinel sits on the same line as the last src line.

Fix: before composing the block, ensure `src` ends with `\n`:
```go
if !strings.HasSuffix(src, "\n") {
    src += "\n"
}
```

#### 9c. `binary_release` unauthenticated GitHub API (7d)

`githubLatestTag` hits `api.github.com` with no auth. Anonymous rate limit is 60 req/hour per IP; on shared CI runners this triggers regularly.

Fix: when `os.Getenv("GH_TOKEN") != ""`, send `Authorization: Bearer $GH_TOKEN`. One-liner change in `githubLatestTag`.

#### 9d. `binary_release` no HTTP timeout (7e)

`http.Get(url)` with no timeout or context. Stalled connections hang the phase indefinitely.

Fix: build an `*http.Client` with timeouts (5 minutes for downloads, 30 seconds for tag lookup). Longer term, thread `context.Context` through `phase.Context` and use `http.NewRequestWithContext`.

#### 9e. `ssh_setup` idempotency gap (7f)

`ShouldRun` only checks that `~/.ssh/id_ed25519` exists and `github.com` is in `known_hosts`. Does not check:
- That the key is registered on GitHub (`github_keys.json` exists).
- That `allowed_signers` contains the local pubkey.

Concretely: if a previous run generated the key but `GH_TOKEN` was unset (so GitHub registration was skipped with a warning), the next run with `GH_TOKEN` set will still skip the whole phase. Workaround today is `--force`.

Fix: also return true (should run) if:
- `GH_TOKEN` is set AND `~/.config/sproot/github_keys.json` does not exist.
- `~/.ssh/allowed_signers` does not contain the local public key.

#### 9f. Sprite `cloneOrPull` does not handle changed `config_repo` URL (7h, low priority)

If the user changes `config_repo` in `~/.sproot/config`, the next setup still fetches from the old remote.

Fix: before fetching, compare `git remote get-url origin` against `opts.ConfigRepo`. If they differ, either re-clone or `git remote set-url origin <new>`.

---

### Phase 10: Binary injection cross-arch fix (CRITICAL BUG from todo.md)

`RunNew` reads its own executable and injects it into the sprite:
```go
execPath, err := os.Executable()
binaryData, err := os.ReadFile(execPath)
handle.WriteFile("/usr/local/bin/sproot", binaryData, 0755)
```

Sprites run Linux/amd64. If sproot is run from a macOS arm64 host, the injected binary is Mach-O arm64 and the sprite gets `exec: Exec format error`.

Fix: embed the Linux/amd64 binary into the release build using `go:embed`, and inject the embedded binary instead of the host binary. The goreleaser build produces `linux/amd64` already; the embed approach requires either:
1. A release-time `embed.FS` populated by the build pipeline (complex).
2. Download the matching Linux/amd64 binary from GitHub releases at `sproot new` time (simpler, requires network, uses version tag).
3. Require the user to run sproot from a Linux host or a sprite (simplest short term; document the limitation).

**OPEN QUESTION**: which approach? The todo entry shows this is a real hit (actual error from the user). Options 2 (download the matching version) or 1 (embed in release build) are the correct long-term fix. Option 3 is a doc-only workaround.

---

### Phase 11: UX improvements

Items from todo.md.

#### 11a. Interactive config init

`sproot config init` currently writes a skeleton file. Make it interactive: prompt for `config_repo`, `token_env`, `gh_token_env`, `default_org`; validate at the end. Add `--non-interactive` flag to preserve current behavior for scripting.

#### 11b. Explain required GitHub token scopes

Add to docs (and to `sproot config init` prompts): minimum scopes for `gh_token_env` are `repo` (cloning private repos) plus `admin:public_key` and `admin:ssh_signing_key` if using `ssh_setup`.

#### 11c. `sproot new` opens console after setup

After `sproot setup` completes, `sproot new` should drop the user into the sprite console. Add `--no-console` flag to opt out.

#### 11d. `sproot console <name>` command

Wrap the sprite console command so users do not need the `sprite` CLI.

#### 11e. `sproot list` command

List only sprites created by sproot (identified via a label set at creation time). Pass `{ "sproot": "true" }` as metadata/labels when calling `client.CreateSprite`. `sproot list` filters by that label.

#### 11f. Auto-setup config when missing

If `sproot` is invoked with no `~/.sproot/config`, offer to run `config init` interactively before the command proceeds.

#### 11g. Debug logging flag

Add `--debug` global flag that enables verbose logging throughout (phase runner decisions, HTTP requests, command execution). Use `pkg/log` level system or a context value.

#### 11h. Validate `sproot.yaml` before creating sprite

`sproot new` currently validates the host config but does not read or validate `sproot.yaml` before spending API quota creating a sprite. If the config file is broken, the sprite is created but setup fails.

Fix: in `RunNew`, shallow-clone the config repo (or accept a `--config-path` override pointing to a local file), load and validate `sproot.yaml`, then proceed to sprite creation. The `--dry-run` flag already skips sprite creation; this adds pre-flight validation even on live runs.

#### 11i. `sproot validate` also validates `sproot.yaml` reachability

Currently validates syntax only. Optionally warn if referenced `src` paths in `file_template`/`rc_block` do not exist at the config repo root.

---

### Phase 12: Sprite wrapping and SDK alignment

#### 12a. Wrap more sprite CLI commands

Review the sprites-go SDK and expose useful subcommands so users do not need the `sprite` CLI separately. Minimum set: `console`, `list`, `status`. Stretch: `checkpoint`, `restore`, `exec`.

#### 12b. Remove things not supported by sprites-go SDK

Audit each place sproot calls the SDK. If a feature is documented but not present in the installed SDK version, either remove the call or gate it on a runtime check.

#### 12c. Checkpointing integration

`sproot.yaml` should be able to declare `checkpoint_after_setup: true`. After setup phases complete, `sproot new` creates a checkpoint. Doing it before setup is pointless (checkpoint of the base image only). Expose `sproot checkpoint <name>` and `sproot restore <name>` wrappers.

#### 12d. Accessible setup state from host

`sproot status <name>` already streams the in-sprite state table. Improve: show a summary of what is installed vs. outstanding, surface any phase errors, and make it readable without connecting to the sprite (cache state on the host or expose it via sprite filesystem).

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

| # | Area | Question | Blocks |
|---|------|----------|--------|
| Q1 | `env` block | Implement, drop, or mark as future? | 8i, arch of env forwarding |
| Q2 | `file_template` | Add `Template bool` opt-in or document always-attempt? | 8a (fixing the example anyway) |
| Q3 | `cmd` `name` field | Add field or drop from docs? | 8a (fixing the example anyway) |
| Q4 | `binary_release` checksums | Add optional `checksum:` field or known tradeoff? | 9d |
| Q5 | validate commands | Keep `config validate` and `validate` separate (recommended) or combine? | 8b |
| Q6 | Cross-arch binary injection | Embed Linux/amd64 binary, download it at runtime, or document limitation? | Phase 10 |
| Q7 | Multi-target backward compat | Flat `phases:` in Phase 13a treated as implicit default target? | Phase 13a |

---

## Suggested order of execution

Once open questions are answered:

1. **Phase 8 doc fixes** (no questions blocking 8b-8j): 8b, 8c, 8d, 8e, 8f, 8g, 8h, 8i, 8j.
2. **Phase 8a** (YAML format fix for all 17 modules): biggest user-visible doc impact.
3. **Phase 9 bug fixes** (no questions blocking): 9a, 9b, 9c, 9d, 9e, 9f.
4. **Phase 10** (cross-arch injection fix): depends on answer to Q6; affects all non-Linux users.
5. **Question-dependent changes**: Q2 (file_template), Q3 (cmd name), Q1 (env block), Q4 (binary_release checksums).
6. **Phase 11 UX** (11a through 11i): independent; can be picked off one at a time.
7. **Phase 12 SDK alignment** (12a-12d): review SDK docs first.
8. **Phase 13 multi-target and push** (13a-13c): architecture changes; coordinate with Phase 6 (sprite config repo).
9. **Phase 14 intelligence** (14a-14c): after everything else is stable.

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
