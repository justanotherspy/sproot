# sproot

sproot bootstraps [sprite.dev](https://sprite.dev) sprites from a user-owned config repo. It replaces bash-based setup scripts with a single, versioned Go binary driven by a `sproot.yaml` file you control.

## How it works

```
sproot new my-sprite
```

1. Reads `~/.sproot/config` for your config repo URL and SSH key path
2. Creates a new sprite via `sprite create`
3. Copies your SSH key and the sproot binary into the sprite
4. Runs `sproot setup` inside the sprite, which clones your config repo and executes each phase

Each phase is idempotent. Re-running setup is safe.

## Config repo

Your config repo holds a `sproot.yaml` that lists the phases to run:

```yaml
schema_version: 1

identity:
  git_user_name: "Your Name"
  git_user_email: "you@example.com"
  gh_username: yourname

phases:
  - type: apt
    packages: [shellcheck, jq]
  - type: uv_tool
    tools: [{name: ruff}]
  - type: file_template
    src: files/statusline.py
    dest: ~/.claude/statusline.py
    mode: "0755"
  - type: rc_block
    src: files/rc_additions.sh
  - type: repo_clone
    base_dir: ~/repos
    repos:
      - yourname/yourrepo
```

See [docs/modules.md](docs/modules.md) for all module types.

## Host config

```
~/.sproot/
├── config           # YAML: config_repo, private_key_path
└── private/
    └── id_ed25519   # SSH key loaded into each sprite
```

Initialize with:

```
sproot config init
```

## Commands

| Command | Where | Description |
|---------|-------|-------------|
| `sproot new <name>` | host | Create and provision a sprite |
| `sproot destroy <name>` | host | Destroy a sprite |
| `sproot status <name>` | host | Show setup state for a sprite |
| `sproot config init` | host | Write a skeleton ~/.sproot/config |
| `sproot config validate` | host | Validate host config and sproot.yaml |
| `sproot setup` | sprite | Clone config repo and run phases |
| `sproot setup --status` | sprite | Print phase state table |

## Installation

Binaries are available on the [releases page](https://github.com/justanotherspy/sproot/releases).

**One-line install (Linux and macOS):**

```sh
curl -fsSL https://raw.githubusercontent.com/justanotherspy/sproot/main/install.sh | sh
```

**From source:**

```sh
git clone https://github.com/justanotherspy/sproot
cd sproot
make install
```

## Development

```sh
make build        # build ./sproot
make test         # run tests
make check        # vet + test + lint
make lint         # golangci-lint
```

Requires Go 1.25+.

## License

MIT
