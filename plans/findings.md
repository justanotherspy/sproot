# sproot doc and code review findings

Handoff document for Claude Code. The reviewer (an external Claude session) went through `docs/modules.md`, `README.md`, `CLAUDE.md`, and cross-checked them against the code in `internal/`, `cmd/`, and the testdata.

Findings are grouped by severity. Questions for the human are flagged with `OPEN QUESTION` and must be answered before acting on items that depend on them.

Project convention reminder: no emdashes anywhere. Use parens or commas.

---

## 1. CRITICAL: `docs/modules.md` YAML examples are wrong for all 17 modules

### Problem

Every module example in `docs/modules.md` shows a nested structure with the type name as a sub-key:

```yaml
- type: apt
  apt:
    packages:
      - git
```

The actual code does not parse this. `PhaseConfig.UnmarshalYAML` in `internal/config/schema.go` decodes the whole phase node directly into the concrete config struct:

```go
case "apt":
    p.Apt = &AptConfig{}
    return value.Decode(p.Apt)   // decodes the whole node into AptConfig
```

So the real shape is flat:

```yaml
- type: apt
  packages: [shellcheck, jq]
```

### Cross-check

Flat form is used by every other source in the repo:
- `README.md` examples
- `internal/config/testdata/sproot.yaml`
- `testdata/integration/sproot.yaml`
- `plans/sproot.md` examples
- Doc comments inside every module `.go` file (e.g. `internal/phase/modules/apt.go`)

A user following `modules.md` will get either empty configs (the nested sub-key has no matching field, so values stay at zero values) or validation errors like `phases[0] (rc_block): src is required`.

### Action

Rewrite every YAML block in `docs/modules.md` to drop the nested key. Use the flat form already in `internal/config/testdata/sproot.yaml` as the source of truth.

Affected modules (all 17): `apt`, `uv_tool`, `go_install`, `cargo_install`, `binary_release`, `corepack`, `rust_components`, `docker`, `sprite_service`, `git_identity`, `ssh_setup`, `gh_token`, `file_template`, `rc_block`, `repo_clone`, `claude_settings`, `cmd`.

The `env` block at the top is already shown in flat form correctly; only the per-phase examples need the fix.

---

## 2. `docs/modules.md`: `file_template` `template: true` field does not exist

### Problem

The docs claim:

> - `template`: optional; if true, executes `src` as a Go template with `ctx.Identity` as data

`FileTemplateConfig` (`internal/config/schema.go`) has no `Template` field. `render()` in `internal/phase/modules/file_template.go` always attempts `text/template.Parse` and falls back to a literal copy if parsing fails:

```go
tmpl, err := template.New("").Parse(string(raw))
if err != nil {
    // Not a template or parse error; treat file as literal.
    return raw, nil
}
```

This means a literal file containing `{{ .GitUserName }}` (unintentional) will be substituted; a literal file containing `{{` that does not parse will silently fall through to literal.

### `OPEN QUESTION` for user

Pick one:

1. **Document the actual behavior:** templating is always attempted, fallback to literal on parse error, no flag needed. Lower effort. Mildly fragile (the silent substitution risk above).
2. **Add a real `Template bool` field** to `FileTemplateConfig`, gate the template parse on it, and have the docs match. Safer default behavior. Recommended.

### Action (after question is answered)

- If option 1: edit `docs/modules.md` `file_template` section to remove the `template:` field and describe the actual auto-detect behavior including caveats.
- If option 2: add `Template bool \`yaml:"template"\`` to `FileTemplateConfig`. Update `render()` to only call `template.Parse` when `p.cfg.Template` is true. Update or add tests in `internal/phase/modules/file_template_test.go`. Leave docs as-is.

---

## 3. The `env` block is documented but not implemented

### Problem

`docs/modules.md` describes an `env block (top-level)` that "declares host environment variables to forward into the sprite before `sproot setup` runs". Schema, parsing, and validation exist for it (`EnvVar` struct in `internal/config/schema.go`, validation in `internal/config/validate.go`).

But:

- `internal/host/new.go` only ever forwards `GH_TOKEN` from `cfg.GHTokenEnv`. It does not read the `env` block.
- The host never loads `sproot.yaml` at all; only the sprite does, after cloning the config repo.
- `Required: true` is never enforced anywhere.
- Nothing in the codebase reads the parsed `EnvVar` values.

So the `env` block is declarative-only: it parses, it validates, and then nothing happens with it. `gh_token_env` in `~/.sproot/config` does the actual `GH_TOKEN` forwarding.

The docs also lean on this in two other places:
- `ssh_setup` says: "Registers the public key with GitHub ... using `GH_TOKEN` (set via the `env` block)"
- `gh_token` says: "Reads `GH_TOKEN` from the environment (injected via the `env` block in `sproot.yaml`)"

Both are wrong; `GH_TOKEN` arrives via `gh_token_env` on the host config.

### `OPEN QUESTION` for user

Pick one:

1. **Implement it.** `RunNew` shallow-clones the config repo on the host (or has the user provide a path), reads `sproot.yaml`, resolves each `from` env var via `os.Getenv`, fails fast when `required: true` and unset, builds the env slice for the sprite command. Most flexible, biggest change.
2. **Drop it.** Remove `EnvVar`, the `Env` field on `SprootConfig`, the validation cases, the test data, and the docs section. The `gh_token_env` host config field already covers the common case.
3. **Document as future / not yet wired.** Leave parsing in place, mark it as planned, add a TODO in `schema.go`. Note in `docs/modules.md` that the block currently parses but does nothing. Least satisfying.

### Action (after question is answered)

If option 1, also update the `ssh_setup` and `gh_token` doc sections to reference the env block accurately. If option 2 or 3, update both of those sections to say `GH_TOKEN` is forwarded via `gh_token_env` in `~/.sproot/config`.

---

## 4. `docs/modules.md`: `cmd` module `name` field does not exist

### Problem

The docs show:

```yaml
- type: cmd
  cmd:
    run: "..."
    check: "..."
    name: "Install mytool"
```

(That nested form is also wrong per finding 1.)

`CmdConfig` has no `Name` field. `cmdPhase.Name()` returns the literal string `"cmd"` regardless of config.

### `OPEN QUESTION` for user

Pick one:

1. **Add the field.** `Name string \`yaml:"name"\`` on `CmdConfig`. `cmdPhase.Name()` returns `fmt.Sprintf("cmd(%s)", p.cfg.Name)` when set, else `"cmd"`. Matches the disambiguator pattern used by `binary_release(cosign)` and `file_template(<dest>)`.
2. **Drop from docs.**

### Action (after question is answered)

If option 1: schema change, update `cmdPhase.Name()`, add a test case. Docs already mention the field (after fix in finding 1).
If option 2: remove `name:` from the cmd example and the field list in `docs/modules.md`.

---

## 5. README.md issues

### 5a. Commands table is wrong and incomplete

Current row:

| `sproot config validate` | host | Validate host config and sproot.yaml |

This is wrong. `sproot config validate` (see `cmd/sproot/config.go`) only validates `~/.sproot/config`. There is a separate top-level `sproot validate [--path PATH]` command (see `cmd/sproot/validate.go`) that validates `sproot.yaml`. CI uses this one (`ci.yml` validate job).

**Action:**

- Fix description: `sproot config validate` -> "Validate `~/.sproot/config`".
- Add a row: `sproot validate [--path PATH]` (host) "Validate a sproot.yaml file. Defaults to `config_path` from `~/.sproot/config`, or `sproot.yaml` in cwd."

### 5b. "How it works" step 2 is misleading

Current:

> Resolves tokens from your environment (`FLY_API_TOKEN`, `GITHUB_TOKEN`, etc.)

These env var names are not hardcoded. They are whatever the user puts in `token_env` and `gh_token_env`. The skeleton from `RunConfigInit` defaults to `SPRITE_TOKEN`, not `FLY_API_TOKEN`.

**Action:** reword to "Resolves the API and GitHub tokens from the env vars named in `~/.sproot/config` (`token_env` and `gh_token_env`)."

### 5c. Host config example missing `config_path`

`HostConfig` has a `ConfigPath` field (path to the config file within the repo, defaults to `sproot.yaml`). The README example does not show it. The skeleton from `RunConfigInit` also omits it.

**Action:**

- Add to the README host config example:
  ```yaml
  config_path: ""             # optional; defaults to sproot.yaml at the repo root
  ```
- Optionally add the same line to `configSkeleton` in `internal/host/config.go`.

---

## 6. CLAUDE.md issues

### 6a. Phase plan table is stale

Current table shows Phase 7 (Release pipeline) and Phase 8 (Docs) as not done. But:

- `.goreleaser.yaml`, `.github/workflows/release.yml`, `install.sh` all exist (Phase 7 work is at least partly shipped).
- `docs/modules.md` and `README.md` exist (Phase 8 work is shipped, though buggy per this review).

Phase 6 (Convert sprite into config repo) is in a different repo so its status is not visible here.

**Action:** mark Phases 7 and 8 as "partial" or "done" as appropriate. Optionally add a note: "Phase 8 doc accuracy issues are tracked in the findings handoff."

### 6b. CI section undercounts jobs

Current:

> Two jobs run on every push: `build-and-test` and `lint`.

Actual `ci.yml` has three: `build-and-test`, `validate` (runs `./sproot validate --path internal/config/testdata/sproot.yaml`), and `lint`. Plus `integration.yml` adds a build job and six matrix integration tests (only for the repo owner).

**Action:** update the CI section to list all three `ci.yml` jobs and mention the integration workflow.

### 6c. Directory layout is incomplete

Missing `docs/` and `testdata/integration/`.

**Action:** add to the directory layout block:

```
docs/                - user-facing docs (modules.md, etc.)
testdata/integration/ - integration test config used by integration.yml
```

---

## 7. Code bugs

These are real behaviour bugs, not just doc issues. Each can land independently of the doc fixes.

### 7a. `rc_block` `ShouldRun` only checks `.bashrc`

`internal/phase/modules/rc_block.go`:

```go
func (p *rcBlockPhase) ShouldRun(ctx *phase.Context) (bool, error) {
    ...
    bashrc := filepath.Join(home, ".bashrc")
    existing, err := os.ReadFile(bashrc)
    if err != nil {
        return true, nil
    }
    wantHash := blockHash(src)
    return extractBlockHash(string(existing)) != wantHash, nil
}
```

`Run` writes both `.bashrc` and `.zshrc`. `Verify` checks both. So if `.bashrc` has the correct block but `.zshrc` is missing or stale (a user wiped it between runs), `ShouldRun` returns false, `Run` is skipped, then `Verify` fails on `.zshrc`.

**Action:** iterate `.bashrc` and `.zshrc` in `ShouldRun`. Return true if either is missing or has the wrong hash. Add a test for the case where only one file is stale.

### 7b. `rc_block` trailing newline not guaranteed

`applyRCBlock` formats the block as:

```go
block := fmt.Sprintf("\n%s\n%s%s\n", rcBegin, src, rcEnd)
```

If `src` does not end in `\n`, the end sentinel sits on the same line as the last src line. The hash check still passes because both sides see the same body, but it is ugly.

**Action:** ensure `src` ends with `\n` before composing the block. One line:

```go
if !strings.HasSuffix(src, "\n") {
    src += "\n"
}
```

### 7c. `binary_release`: no checksum or signature verification

`internal/phase/modules/binary_release.go`. `downloadAsset` pulls the artifact and `dpkg -i` / extract / chmod-and-copy it with zero verification. sproot itself ships with cosign keyless signing, so the contrast here is sharp: every `binary_release` phase silently runs whatever GitHub serves.

**`OPEN QUESTION` for user:** is this lacking-checksum behavior a known tradeoff, or should it be fixed now?

**Action (if fixing):** add optional `checksum:` field (sha256 string) or `checksum_asset:` (template pointing to a `*_checksums.txt`-style file). Verify before installing. Update `BinaryReleaseConfig`, the templating in `templateAsset`, the run flow, and docs.

### 7d. `binary_release`: unauthenticated GitHub API

`githubLatestTag` hits `https://api.github.com/...` with no auth. The anonymous rate limit is 60 req/hour per IP.

**Action:** when `os.Getenv("GH_TOKEN") != ""`, send `Authorization: Bearer $GH_TOKEN` on the request. Trivial change.

### 7e. `binary_release`: no HTTP timeout or context

`http.Get(url) //nolint:noctx`. Stalled connections can hang the phase indefinitely.

**Action:** build an `*http.Client` with a sane timeout (e.g. 5 minutes for downloads, 30 seconds for the latest-tag lookup), or thread `context.Context` through `phase.Context` and use `http.NewRequestWithContext`. The latter is the cleaner long-term shape.

### 7f. `ssh_setup` idempotency is too narrow

`ShouldRun` only checks that `~/.ssh/id_ed25519` exists and `github.com` is in `known_hosts`. It does not verify that:

- The key is registered with GitHub.
- `allowed_signers` contains the entry.
- The key IDs file at `~/.config/sproot/github_keys.json` exists.

Concretely: if the local key was generated but `GH_TOKEN` was unset on a previous run (so GitHub registration was skipped with a warning), the next run with `GH_TOKEN` set will still skip the whole phase. Workaround today is `--force`.

**Action:** make `ShouldRun` also return true if:

- `GH_TOKEN` is set AND `~/.config/sproot/github_keys.json` does not exist (means we never registered).
- `allowed_signers` does not contain the local public key.

Optionally also fetch GitHub's reported keys and compare fingerprints, but that adds an HTTP call to every `ShouldRun`; probably not worth it.

### 7g. `gh_token` doc lists scopes under the wrong module

`docs/modules.md` says under `gh_token`:

> **Requires:** `GH_TOKEN` set in the sprite environment via the `env` block. Required scopes: `admin:public_key`, `admin:ssh_signing_key`.

Those two scopes are actually needed by `ssh_setup` (it calls the GitHub keys API). `gh_token` itself only needs whatever scopes the user wants `gh` to use (typically `repo`).

**Action:** move the "Required scopes" line to the `ssh_setup` section, or duplicate it in both. While there, also fix the `env block` reference per finding 3.

### 7h. Sprite `cloneOrPull` does not handle changed `config_repo` URL

`internal/sprite/setup.go` does `git fetch && git checkout ref` if the dest dir exists. If the user changes `config_repo` in `~/.sproot/config`, the next setup still fetches from the OLD remote because the cloned dir's `origin` is unchanged.

Edge case, probably low priority.

**Action (low priority):** detect URL drift before fetching. Either re-clone or `git remote set-url origin <new>`. Or just delete the dest dir if `git remote get-url origin` differs from `opts.ConfigRepo`.

### 7i. `gh_token` actual flags differ slightly from docs

Docs say: `gh auth login --with-token --git-protocol ssh`.
Code calls: `gh auth login --hostname github.com --git-protocol ssh --with-token`.

Functionally equivalent. Tiny doc drift.

**Action:** add `--hostname github.com` to the docs string, or describe the flags in prose without the example. Nit-level.

---

## 8. Smaller doc nits (low priority)

- `plans/sproot.md` "Host file layout" still shows `private/id_ed25519` from the abandoned host-key model. The text immediately below already describes the env-var-name model. User has flagged plans as out of date; safe to leave, but worth a note in the file or a "superseded" marker on that block.
- `docs/modules.md` `rc_block` section shows the sentinels as `# BEGIN SPROOT MANAGED BLOCK` / `# END SPROOT MANAGED BLOCK`. This matches the code (`rcBegin` / `rcEnd` in `rc_block.go`). No action.
- `docs/modules.md` `claude_settings` example uses `claude-haiku-4-5-20251001` as an example model. That model exists (it's current). No action.

---

## 9. Open questions summary

Quick rollup of everything tagged `OPEN QUESTION`:

1. **`env` block:** implement, drop, or document as future? (Affects finding 3, plus the `ssh_setup` and `gh_token` doc references.)
2. **`file_template` `template:` flag:** add an opt-in field, or document the always-attempt behavior? (Affects finding 2.)
3. **`cmd` module `name` field:** add the field, or drop from docs? (Affects finding 4.)
4. **`binary_release` checksum verification:** known tradeoff, or fix now? (Affects finding 7c.)
5. **`sproot config validate` vs `sproot validate`:** keep them separate (just fix the README description, per 5a), or combine into one command that checks both files?

---

## 10. Suggested order of execution for Claude Code

Once questions are answered:

1. Quick wins, no questions blocking: 5a, 5b, 5c, 6a, 6b, 6c, 7b, 7d, 7e, 7g, 7h, 7i.
2. Module YAML format fix (1): biggest doc impact, mechanical change.
3. `rc_block` `ShouldRun` fix (7a) plus its test.
4. `ssh_setup` idempotency (7f) plus tests.
5. Whichever way the questions land: 2 (file_template), 3 (env block), 4 (cmd name), 7c (binary_release checksums).

After all changes, run `make check` and the CI `validate` step locally.
