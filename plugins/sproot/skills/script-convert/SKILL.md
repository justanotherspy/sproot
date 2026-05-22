---
name: script-convert
description: Convert a setup/install/bootstrap bash script into a sproot.yaml config plus companion files. Use when the user wants to migrate a shell setup script to sproot, turn a script into sproot modules, or asks "convert this script to sproot".
argument-hint: "[path-to-setup-script]"
allowed-tools: Read, Write, Grep, Glob, Bash
---

# Convert a bash setup script to sproot.yaml

You translate an existing setup/bootstrap bash script into a sproot config: a `sproot.yaml`
plus any companion files it references, ready to drop into a sproot config repo and pass
`sproot validate`.

## Load the reference first

Before converting, read these bundled references (resolve via `${CLAUDE_PLUGIN_ROOT}/reference/`,
or relative to this skill at `../../reference/`):

- `reference/module-map.md`: the bash-idiom -> module decision table and rules.
- `reference/module-schema.md`: exact field names for every module and the top-level shape.

These are load-bearing. Field names and the flat phase structure must match exactly or the output
will not validate.

## Procedure

1. **Ingest.** Take the script from `$1` (a path) or pasted content. Read it in full. If neither
   was given, ask for the script.

2. **Segment** into ordered steps. Split on newlines, `;`, and `&&`; join `\`-continued lines;
   capture heredocs (`<<EOF ... EOF`) as a single file-write step; note functions and simple
   conditionals. Ignore comments, `set -e`, `cd`, and pure echo/logging lines. Strip leading
   `sudo`.

3. **Classify** each step against `reference/module-map.md`. Pick the most specific structured
   module; fall back to `cmd` only when nothing fits.

4. **Coalesce.** One `apt` phase for all packages (with `symlinks` for bat/fd quirks), one
   `repo_clone` for all clones, one `rc_block` companion for all `>> ~/.bashrc|.zshrc|.profile`
   appends.

5. **Companion files.** For file writes (`cat >`/`tee`/heredoc/`printf >`/`cp`) emit a
   `file_template` with `src: files/<name>` and `dest:` the target path; write the body to
   `files/<name>` in the output directory. Map a nearby `chmod 0NNN` to `mode: "0NNN"`. For shell
   rc appends, write all accumulated lines to `files/rc_additions.sh` and emit one `rc_block`.

6. **Secrets.** Never inline a token or secret value. A literal secret, or use of
   `$GH_TOKEN`/`$*_TOKEN`/`$*_SECRET`/`$*_KEY`, becomes a top-level `env:` entry (`from`/`as`) and
   goes on the review list. `gh auth`/SSH-key steps imply `gh_token`/`ssh_setup` plus an `env`
   entry for the PAT.

7. **Order & dependencies.** Preserve the order of distinct steps (install before use). If a
   `binary_release` uses a `cosign:` block, place the phase that installs cosign earlier.

8. **Scaffold.** Emit `schema_version: 1`, then a required `identity:` block with clearly-marked
   placeholders the user must fill in:
   ```yaml
   identity:
     git_user_name: "CHANGE ME"
     git_user_email: "changeme@example.com"
     git_default_branch: main
     gh_username: CHANGE_ME
   ```
   Then the `env:` block (if any) and the `phases:` list. Drop `git config user.name/email` and
   default-branch steps: those come from `identity` (note this in the summary).

9. **Write output.** Ask where to write (default: `./sproot-out/`). Create
   `<dir>/sproot.yaml` and `<dir>/files/...`. Mirror the YAML style of
   `internal/config/testdata/sproot.yaml` (two-space indent, flat phase fields).

10. **Validate.** Build sproot if a binary is not already present
    (`go build -o /tmp/sproot ./cmd/sproot` from the repo root, or `make build`), then run
    `<sproot> validate --path <dir>/sproot.yaml`. Fix any reported errors and re-run until it
    passes. If you are not inside the sproot repo and cannot build, tell the user to run
    `sproot validate --path <dir>/sproot.yaml` themselves.

11. **Report.** Print a conversion summary table: each source step -> chosen module (or "dropped",
    with reason). Then a **Needs review** list covering: secrets and the `env` entries made for
    them, interactive prompts (`read`, `apt` without `-y`), host-specific absolute paths,
    `curl|sh` network installers, and every `cmd` fallback with the reason it could not be
    structured.

## Worked examples

`examples/<case>/` pairs an `input.sh` with its `expected/` output (`sproot.yaml` + `files/`).
They are the golden fixtures CI validates; use them as patterns. Cases include: `apt-and-symlinks`,
`language-tools`, `binary-release`, `dotfiles`, `repos-and-git`, `cmd-fallback`,
`secrets-and-services`.

## Reminders

- All phase fields are flat (e.g. `type: corepack` then `managers: [...]`, never nested).
- No emdashes in generated output (a sproot project convention): use parens or commas.
- When unsure between a structured module and `cmd`, prefer the structured module if the
  mapping is unambiguous; otherwise use `cmd` with a `check:` and flag it.
