# Bash idiom -> sproot module map

The decision table for converting setup scripts. For each recognized bash idiom, pick the most
specific module. Fall back to `cmd` only when nothing structured fits. Field shapes are in
[module-schema.md](module-schema.md).

## Decision table

| Bash idiom (after dropping `sudo`) | Module | Notes |
|---|---|---|
| `apt-get install X` / `apt install X` | `apt` | Collect all packages across the script into one `apt` phase. `bat`->`batcat`, `fd`/`fd-find`->`fdfind`: add `symlinks`. |
| `ln -s SRC DST` right after an apt install | fold into that `apt` phase's `symlinks` | Else use `cmd`. |
| `uv tool install X` | `uv_tool` | `tools: [{name: X}]`; add `pkg:` if the install name differs from the binary. |
| `pipx install X` | `uv_tool` | Same role as pipx; prefer `uv_tool`. |
| `pip install X` / `pip3 install X` for a CLI tool | `uv_tool` (preferred) | If it is a library (not a CLI), or installs into a project venv, use `cmd`. |
| `go install pkg@ver` | `go_install` | `tools: [{pkg, version}]`. Default `version: latest` if `@latest` or none given. |
| `cargo install X` | `cargo_install` | Map `--locked`->`locked: true`, `--features a,b`->`features: [a, b]`, `--version V`->`version: "V"`. |
| `rustup component add X [Y]` | `rust_components` | `components: [X, Y]`. |
| `corepack enable` / `corepack prepare pnpm --activate` | `corepack` | `managers:` = the managers enabled (default `[pnpm]` if only `corepack enable`, otherwise list what is activated). |
| `npm install` / `npm ci` (in a project dir) | `npm` | `dir:` = the directory with `package.json`. |
| `npm install -g X` (global) | `cmd` | No structured global-npm module. `run: "npm install -g X"`, `check: "command -v X"`. |
| download a GitHub release asset (curl/wget of `github.com/<o>/<r>/releases/...`, or `gh release download`) | `binary_release` | Parse `repo`, `asset` (template the version/arch), `install` (`dpkg` for `.deb`, `tar+install` for tarballs, `raw` for a single binary). See arch tokens in module-schema. |
| `curl ... \| sh` / `wget ... \| bash` installer (NOT a GitHub release asset) | `cmd` | `run:` the pipeline verbatim; add a `check:` on the resulting binary. Flag for review (network installers are opaque). |
| `git clone URL [DIR]` | `repo_clone` | `github.com/o/r` -> short form `o/r` under `base_dir`. Other hosts or explicit dirs -> long form `{url, dest}`. Group all clones into one phase. |
| `git config --global user.name/user.email/init.defaultBranch` | top-level `identity` (drop the command) | These come from the `identity` block; do not emit a phase for them. |
| other `git config --global K V` | `git_identity` with `config: {K: "V"}` | Quote values as strings. |
| `cat > F <<EOF ... EOF`, `tee F <<EOF`, `printf ... > F`, `cp SRC F` (writing a static file) | `file_template` + companion file | Extract the body to `files/<name>`, set `dest: F`. Map a nearby `chmod 0NNN F` to `mode: "0NNN"`. |
| `echo ... >> ~/.bashrc` / `>> ~/.zshrc` / `>> ~/.profile` (shell init) | `rc_block` + companion `files/rc_additions.sh` | Accumulate ALL such appends into ONE rc_block companion file. |
| install docker (`get.docker.com`, `apt install docker-ce`) | `docker` | If the script writes `/etc/docker/daemon.json`, fold that JSON into `daemon_json:`. |
| start a long-running daemon/service (`dockerd &`, `systemctl start`, `nohup ... &`, a `start.sh`) | `sprite_service` | `cmd:` = the executable; `args:`/`http_port:`/`needs:` as applicable. Sprites have no systemd. |
| configure Claude Code (`claude upgrade`, write `~/.claude/settings.json`, write a global `CLAUDE.md`) | `claude` | `upgrade: true`, `settings:` (the JSON), `claude_md:` (companion file). |
| `gh auth login` with a token | `gh_token` + an `env` entry for the token | Never inline the token. |
| generate/register an SSH key with GitHub | `ssh_setup` + an `env` entry for the PAT | |
| anything unrecognized | `cmd` | `run:` verbatim; add a best-effort `check:`; set a short `name:`. Flag for review. |

## Rules

1. **Drop `sudo`** prefixes. Modules escalate privilege themselves where needed.
2. **Coalesce**: one `apt` phase for all packages, one `repo_clone` for all clones, one
   `rc_block` companion file for all shell-rc appends.
3. **Preserve order** of distinct steps; install-before-use matters. If a `binary_release` uses a
   `cosign:` block, the phase that installs `cosign` must come earlier in the list.
4. **Secrets never inline.** A literal token, or use of `$GH_TOKEN`/`$*_TOKEN`/`$*_SECRET`/
   `$*_KEY`, becomes a top-level `env:` entry (`from`/`as`) and goes on the review list.
5. **Don't emit phases for things sproot already does**: `git config user.name/email`,
   default-branch config (covered by `identity`), and re-installing tools the base image has.
6. **Prefer structured modules** over `cmd`. Reach for `cmd` only when no module fits, and always
   try to give it a `check:` for idempotency.
7. **Idempotency `check:` patterns** for `cmd`: `command -v <bin>` for installed binaries,
   `test -f <path>` for created files, `test -d <dir>` for directories, `grep -q <marker> <file>`
   for edited files.

## Always flag for human review

- secrets/tokens, and the `env` entries created for them;
- interactive prompts (`read`, password prompts, `apt` without `-y`);
- host-specific absolute paths that may not exist in a sprite;
- `curl|sh` network installers (opaque, unversioned);
- anything that fell back to `cmd` and why.
