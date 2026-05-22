# sproot doc and code review findings (round 2)

Handoff document for Claude Code. A fresh review of the sproot repo on main, cross-checking `docs/`, `plans/`, `README.md`, `CLAUDE.md`, `MIGRATION.md`, and the code in `internal/`, `cmd/`, and the testdata against each other.

The previous findings handoff (`plans/findings.md`) is fully superseded; every item there was resolved in Phase 8 or Phase 9. The items below are new.

Findings are grouped by severity. Open questions for the human are flagged with `OPEN QUESTION`.

Project convention reminder: no emdashes anywhere. Use parens or commas.

---

## 1. CRITICAL: `sproot push` silently drops env block and `GH_TOKEN` forwarding

### Problem

Compare the env handling in the two code paths that run `sproot setup` inside a sprite.

`internal/host/new.go` (in `RunNew`):

```go
var env []string
if ghToken != "" {
    env = []string{"GH_TOKEN=" + ghToken}
}
env = append(env, envBlock...)
if err := handle.RunCommand("sproot", args, env, os.Stdout, os.Stderr); err != nil {
    return err
}
```

`internal/host/push.go` (in `pushOne`):

```go
if err := handle.RunCommand("sproot", args, nil, stdout, stderr); err != nil {
    return err
}
```

`push.go` passes `nil` for env. Consequences:

- The host config `gh_token_env` (implemented in Phase 8 Q1) is forwarded by `sproot new` but ignored by `sproot push`.
- The `env:` block in `sproot.yaml` is resolved by `currentConfigSHA` (which calls `readEnvBlock` and gets back the env slice) but then thrown away in `pushOne`. The slice never reaches `RunCommand`.
- `Required: true` env vars are still enforced (because `readEnvBlock` fails fast) but the resolved values do not propagate.
- Any user-defined env vars (e.g. `DATABASE_URL`, `STRIPE_KEY`) do not reach phases on push.
- `ssh_setup` and `gh_token` running under `--force` (push always sets --force) will not have `GH_TOKEN` available. Both currently degrade gracefully (warning instead of error), so push does not crash, but key registration is silently skipped and `gh auth login` is bypassed.

### Cross-check

- `internal/host/new_test.go` has `TestRunNew_EnvBlockForwarded` and `TestRunNew_InjectsBinaryAndForwardsGHToken`. Both confirm new.go forwards env correctly.
- `internal/host/push_test.go` has no equivalent test. There is no assertion about `handle.lastCmdEnv` anywhere in push tests.
- Reading `pushOne` end to end confirms no env slice is built.

### Action

1. Plumb the env slice through `pushOne`. `RunPush` already needs `readEnvBlock` output for the SHA, so capture both halves:
   - Change `shaFn` signature (or add a sibling) so it returns the env slice alongside SHA and meta.
   - Or, more cleanly, give `pushOne` access to the host config (`cfg`) and the parsed `sprootCfg`, and let it build the env slice the same way `new.go` does.
2. The simpler refactor: extract the env-building from `new.go` into a helper (e.g. `buildSpriteEnv(ghToken string, envBlock []string) []string`) and call it from both places. Note `new.go` resolves `ghToken` from `cfg.GHTokenEnv`; `push.go` already has `cfg` in scope.
3. Add `TestRunPush_ForwardsEnvBlock` and `TestRunPush_ForwardsGHToken` mirroring the new.go tests.
4. Run the `push-and-outdated` integration job locally with an `env:` block in the test config to confirm.

---

## 2. `deleteGHKey` uses `http.DefaultClient` with no timeout

### Problem

`internal/host/destroy.go`:

```go
resp, err := http.DefaultClient.Do(req)
```

Same hazard finding 9d fixed in `binary_release.go`. If GitHub's API hangs, `sproot destroy` hangs indefinitely. Users hitting an outage cannot Ctrl+C cleanly because there is no context wired through either.

The same pattern is present in `internal/phase/modules/ssh_setup.go` `postGHKey`:

```go
resp, err := http.DefaultClient.Do(req)
```

Both should use a bounded client.

### Action

Add a package-level client to each file with a sane timeout (30 seconds is consistent with `tagClient` in `binary_release.go`):

```go
var ghAPIClient = &http.Client{Timeout: 30 * time.Second}
```

Then replace `http.DefaultClient.Do(req)` with `ghAPIClient.Do(req)` in both `destroy.go` (`deleteGHKey`) and `ssh_setup.go` (`postGHKey`).

Optional follow-up: thread `context.Context` through phase.Context so callers can cancel in-flight network operations. Lower priority.

---

## 3. Duplicate "sproot" label constant

### Problem

Two constants hold the same string value in the same package:

- `internal/host/new.go`: `const sprootLabel = "sproot"`
- `internal/host/labels.go`: `labelBase = "sproot"` (inside the const block with the other `labelTarget`, `labelSource`, etc.)

Usage is split inconsistently:
- `new.go`, `list.go`, `push.go`, `outdated.go` all use `sprootLabel` in `hasLabel(e.Labels, sprootLabel)`.
- `labels.go` uses `labelBase` only inside `ConfigMeta.Labels()`.
- `labels_test.go` uses `labelBase`.

### Action

Consolidate to one constant. `labelBase` is the more consistent name with `labelTarget`/`labelSource`/`labelRepo`/`labelRef`/`labelSHA`. Remove `sprootLabel` and update the four files that reference it.

(Trivial mechanical change; safe to bundle with another PR.)

---

## 4. `ConfigSHA` formats full 64-char hex then slices

### Problem

`internal/host/labels.go`:

```go
func ConfigSHA(data []byte) string {
    h := sha256.Sum256(data)
    return fmt.Sprintf("%x", h[:])[:12]
}
```

This formats the entire 64-char hex string then keeps the first 12 chars. Not broken, just wasteful in a function called by `currentConfigSHA` on every `outdated` and `push`.

### Action

```go
return fmt.Sprintf("%x", h[:6])
```

Same output, half the work. Trivial.

---

## 5. `plans/sproot.md` Phase 12b is stale

### Problem

Phase 12b in `plans/sproot.md` claims:

> Added `Upgrade`, `Checkpoint`, `ListCheckpoints`, `Restore` to `SpriteHandle` interface and `realHandle`.
> Added `UpgradeSprite` to `SpritesClient` interface and `realClient`.

Current `internal/host/client.go`:

- `SpriteHandle` interface has: `WriteFile, ReadFile, RunCommand, Console, Checkpoint, ListCheckpoints, Restore, SetLabels`. No `Upgrade`.
- `SpritesClient` interface has: `CreateSprite, GetHandle, DestroySprite, ListSprites`. No `UpgradeSprite`.

Phase 15h explains why ("upgrade was changed to run `sprite upgrade` inside the sprite rather than calling the SDK VM upgrade method"), so the methods were removed when that switch happened. But Phase 12b still reads as if they exist.

### Action

Append a parenthetical to Phase 12b in `plans/sproot.md`:

> Added `Upgrade`, `Checkpoint`, `ListCheckpoints`, `Restore` to `SpriteHandle` interface and `realHandle`. (Note: `Upgrade` was later removed in Phase 15h when `sproot upgrade` switched to running `sprite upgrade` inside the sprite.)
> Added `UpgradeSprite` to `SpritesClient` interface and `realClient`. (Note: also removed in Phase 15h, see above.)

---

## 6. README `sproot push` row missing `--only` flag

### Problem

`cmd/sproot/push.go`:

```go
cmd.Flags().StringVar(&opts.Only, "only", "", "pass --only to sproot setup (run only the named phase type)")
```

README commands table for push:

| `sproot push` | host | Re-run setup on all sproot-managed sprites (`--name` for one, `--target` to select a target, `--no-checkpoint` to skip pre-push checkpoint, `--skip-verify` to skip end-of-run checks) |

Missing: `--only <type>`. CI uses it (`--only cmd` in the `push-and-outdated` job in `.github/workflows/integration.yml`).

### Action

Update the push row to mention `--only`:

> ... (`--name` for one, `--target` to select a target, `--only <type>` to run a single phase type, `--no-checkpoint` to skip pre-push checkpoint, `--skip-verify` to skip end-of-run checks)

---

## 7. `currentConfigSHA` re-clones the git repo on every call

### Problem

`internal/host/push.go`:

```go
func currentConfigSHA(cfg *config.HostConfig, l *log.Logger) (string, ConfigMeta, error) {
    ...
    // Git source: readEnvBlock clones into its own temp dir and returns the SHA.
    l.Debugf("cloning config repo to compute current SHA")
    _, _, sha, err := readEnvBlock(cfg.SprootConfigRepo, cfg.SprootConfigRef, cfg.SprootConfigPath, l)
    ...
}
```

Called by `sproot outdated` and `sproot push`. `outdated` is the kind of command users may run frequently (in scripts, in a watch loop). Each call does `git clone --depth 1` into a fresh temp dir, reads one yaml file, then deletes the dir.

### Action

Low priority, two reasonable options:

1. Cache the SHA + timestamp in `~/.cache/sproot/sha-cache.json` and reuse for some short interval (e.g. 60 seconds, controlled by a `--no-cache` flag).
2. Use `git ls-remote` to get the HEAD commit SHA cheaply; only clone when it differs from a cached commit -> file-SHA mapping.

Neither is urgent since clones are fast. Worth a TODO comment in `currentConfigSHA` so a future maintainer notices.

### `OPEN QUESTION` for user

Is this performance concern worth addressing now, or defer? Defer is the safe answer unless users have complained.

---

## 8. `RunNew` clones the git config repo twice end-to-end

### Problem

For git config sources, `sproot new` clones twice:

1. Host-side clone in `readEnvBlock` (just to resolve env vars and compute SHA).
2. In-sprite clone in `RunSetup` (the real one, runs every phase).

The in-sprite clone is unavoidable (the sprite needs the files). The host clone could be avoided in the common case where the `env:` block is empty.

### Action

Low priority. Possible approaches:

1. Skip the host clone when `cfg.SprootConfigSource != "local"` AND we can know up front there is no env block. We cannot know that without reading the yaml, which requires fetching it. So this is hard to do cleanly without `git archive`.
2. Use `git archive --remote=<url> <ref> sproot.yaml | tar -xO` to fetch only the yaml file. Some servers (notably GitHub over HTTPS) do not support `git archive --remote`. Would have to be SSH-only.
3. Accept the double clone as the price of correctness.

### `OPEN QUESTION` for user

Defer this entirely? It is only relevant if `sproot new` startup time becomes a complaint.

---

## 9. `labels.go` `Labels()` emits empty target values

### Problem

`internal/host/labels.go`:

```go
labels := []string{
    labelBase,
    labelTarget + "=" + m.Target,   // "sproot-target=" when Target is empty
    labelSource + "=" + m.Source,
    labelSHA + "=" + m.SHA,
}
if m.Repo != "" {
    labels = append(labels, labelRepo+"="+m.Repo)
}
if m.Ref != "" {
    labels = append(labels, labelRef+"="+m.Ref)
}
```

`labelTarget` is always emitted, even when `m.Target` is empty (the default-target case). The label list ends up with a `sproot-target=` entry with no value. `ParseConfigMeta` handles this fine, so no behavior bug.

`Repo` and `Ref` are correctly omitted when empty. `Target` is inconsistent.

### Action

Optional cleanup. Move `labelTarget` to the conditional block:

```go
if m.Target != "" {
    labels = append(labels, labelTarget+"="+m.Target)
}
```

Update the tests in `labels_test.go` accordingly. `TestParseConfigMeta_Empty` already expects an empty `Target`, so round-tripping still works.

---

## 10. `plans/findings.md` is fully superseded

### Problem

Every OPEN QUESTION (Q1 to Q5) and every code bug (7a to 7i) in `plans/findings.md` was resolved in Phase 8 or Phase 9 per `plans/sproot.md`. The file still reads as an open handoff.

### Action

Pick one:

1. Add a `> SUPERSEDED` header at the top of `findings.md` pointing at Phase 8 and Phase 9 in `plans/sproot.md`. Keep the file as historical record.
2. Move it to `plans/archive/findings-2026-05.md`.
3. Delete it (the resolutions are already captured in `plans/sproot.md` Phase 8 and Phase 9 summaries).

Recommendation: option 1. Cheapest and preserves history.

---

## 11. `plans/todo.md` "Planned" section has stale Phase 13 entries

### Problem

`plans/todo.md` `Planned (tracked in plans/sproot.md)` section lists these as planned:

- `add a way to pass in a sproot.yaml file from the host instead of a git repo (Phase 13b)`
- `have a sproot push command that pushes changes to all sproot-labeled sprites (Phase 13c)`
- `multi-target support in sproot.yaml with extends (Phase 13a)`

All three are done per `plans/sproot.md` Phase 13.

Phase 14a, 14b, 14c are correctly still listed as planned.

The `inter-sproot URL templating ... (Phase 13a future direction)` is also correctly still planned.

### Action

Remove the three done items from `plans/todo.md`. Move them to a `Done (Phase 13)` section if you want to preserve the trail, similar to the existing `Done (phases 8-12)` section.

---

## 12. `plans/sproot.md` "Host file layout" diagram is partial

### Problem

```
~/.sproot/
└── config           # YAML: config_repo, config_ref, token_env, gh_token_env, default_org
```

This diagram does not mention `sproot_config_source` or `sproot_config_local_path`, which were added in Phase 13b. It also still uses the prefix-less key names (`config_repo`, `config_ref`) but the actual host config now uses `sproot_config_repo`, `sproot_config_ref`, `sproot_config_path`.

The parenthetical immediately below already corrects the abandoned `private/id_ed25519` model, so that part of finding 8 from the old `findings.md` is partially handled.

### Action

Update the inline comment to include the current key set:

```
~/.sproot/
└── config.yaml      # sproot_config_source, sproot_config_repo, sproot_config_ref,
                    # sproot_config_path, sproot_config_local_path, token_env,
                    # gh_token_env, default_org
```

Or just replace the whole "Host file layout" block with a pointer to the `HostConfig` struct doc comment in `internal/config/schema.go`, which is the canonical source.

---

## 13. `MIGRATION.md` "Known cmd workarounds" should be enumerated in plans

### Problem

`MIGRATION.md` lists five concrete `cmd` workarounds tracked for upstream fixes:

- `bat`/`fd` symlinks (apt module should have a `symlinks` field)
- `uv` auto-bootstrap (uv_tool should install uv when absent)
- garlic package/binary mismatch (uv_tool needs `pkg` field separate from binary name)
- `binary_release` arch naming (gitleaks uses `x64`, hadolint uses `x86_64`; needs `{x64_arch}` and `{x86_64_arch}` template variables)
- `docker` `daemon.json` (docker module should support a `daemon_json` config field)

These all map to Phase 15e in `plans/sproot.md` (`update modules to better handle different cases (fewer cmd fallbacks)`), but Phase 15e is currently a one-liner. The specific items are at risk of being lost.

### Action

Expand Phase 15e in `plans/sproot.md` to enumerate the five items as 15e1 through 15e5, each with the specific module and field that needs to change. Cross-reference back to `MIGRATION.md`.

---

## 14. Suggested order of execution for Claude Code

Quick wins (no questions blocking, mechanical):

1. **6** (README push row gets `--only`)
2. **5** (Phase 12b superseded note in plans/sproot.md)
3. **3** (consolidate `sprootLabel` and `labelBase`)
4. **4** (`ConfigSHA` formats only 6 bytes)
5. **9** (omit empty `labelTarget`)
6. **10** (mark findings.md superseded)
7. **11** (prune Phase 13 entries from todo.md)
8. **12** (refresh Host file layout in plans/sproot.md)
9. **13** (enumerate Phase 15e items)

Real fixes with tests:

10. **2** (HTTP timeouts on `deleteGHKey` and `postGHKey`)
11. **1** (push env forwarding) plus `TestRunPush_ForwardsEnvBlock` and `TestRunPush_ForwardsGHToken`

Deferred unless user answers OPEN QUESTION:

12. **7** (currentConfigSHA caching)
13. **8** (avoid double clone in RunNew)

After all changes:

```
make check
./sproot validate --path internal/config/testdata/sproot.yaml
./sproot validate --path internal/config/testdata/sproot_targets.yaml
```

Bundle the quick wins into one PR (label as docs+cleanup), put finding 1 in its own PR (it changes runtime behaviour and needs careful test coverage), put finding 2 in its own PR (cross-cuts two files in different packages).
