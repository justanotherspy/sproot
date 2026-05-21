# Sprite CLI Commands Reference

Source: https://docs.sprites.dev/cli/commands/

Complete reference for all `sprite` CLI commands.

## Installation

```bash
curl -fsSL https://sprites.dev/install.sh | sh
```

## Authentication Commands

### `sprite login`

Authenticate with Fly.io.

```bash
sprite login [flags] [api-url]
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite login
sprite login -o my-org
sprite login https://custom-api.sprites.dev
```

### `sprite logout`

Remove Sprites configuration.

```bash
sprite logout [flags]
```

**Options:**
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite logout
```

### `sprite org auth`

Add an API token.

```bash
sprite org auth [-o <api>:<org>] [api-url|alias]
```

**Aliases:** `sprite orgs`, `sprite organizations`, `sprite o`

**Options:**
- `-o, --org <name>` - Specify organization
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite org auth                                    # Use default API
sprite org auth https://custom-api.sprites.dev     # Custom API URL
sprite org auth prod                               # Use 'prod' alias
sprite org auth -o myorg                           # Specific organization
sprite org auth -o prod:my-org                     # Alias with org override
sprite org auth https://staging-api.sprites.dev -o staging:test-org
```

### `sprite org list`

Show configured tokens.

```bash
sprite org list [api-url|alias]
```

**Options:**
- `-h, --help` - Show this help message

### `sprite org logout`

Remove all tokens.

```bash
sprite org logout [flags]
```

**Options:**
- `--force` - Skip confirmation prompt
- `-h, --help` - Show this help message

### `sprite org keyring disable`

Disable keyring usage.

```bash
sprite org keyring disable
```

### `sprite org keyring enable`

Enable keyring usage.

```bash
sprite org keyring enable
```

### `sprite auth setup`

Set up authentication using a pre-generated token (for CI/CD).

```bash
sprite auth setup --token <token>
```

**Options:**
- `--token <token>` - Token in format: `org-slug/org-id/token-id/token-value`
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite auth setup --token "org-slug/org-id/token-id/token-value"
```

## Sprite Management

### `sprite create`

Create a new sprite.

```bash
sprite create [flags] [sprite-name]
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `--skip-console` - Exit after creating instead of connecting to console
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite create my-sprite
sprite create -o myorg development-sprite
sprite create
```

### `sprite use`

Activate a sprite for the current directory. Creates a `.sprite` file.

```bash
sprite use [flags] [sprite-name]
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `--unset` - Remove the `.sprite` file from current directory
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite use my-sprite
sprite use -o myorg dev-sprite
sprite use
```

### `sprite list`

List all sprites.

```bash
sprite list [flags]
```

**Aliases:** `sprite ls`

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-w, --watch` - Watch for live updates
- `--prefix <prefix>` - Filter sprites by name prefix
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite list
sprite list -o myorg
sprite list --prefix dev
```

### `sprite destroy`

Destroy a sprite permanently.

```bash
sprite destroy [flags] [sprite-name]
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `--force` - Skip confirmation prompt
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite destroy mysprite
sprite destroy -o myorg mysprite
sprite destroy --force mysprite
```

## Command Execution

### `sprite exec`

Execute a command in the sprite environment.

```bash
sprite exec [flags] <command> [args...]
```

**Aliases:** `sprite x`

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `--dir <path>` - Working directory for command
- `--tty` - Allocate pseudo-TTY
- `--env <vars>` - Environment variables (`KEY=value,KEY2=value2`)
- `--http-post` - Use HTTP/1.1 POST instead of WebSockets (non-TTY only)
- `--file <source:dest>` - Upload file before exec (repeatable)
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite exec ls -la
sprite exec --dir /app echo hello world
sprite exec --env KEY=value,FOO=bar env
sprite exec --tty /bin/bash
sprite exec -o myorg -s mysprite npm start
```

### `sprite console`

Open an interactive shell in the sprite environment.

```bash
sprite console [flags]
```

**Aliases:** `sprite c`

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-h, --help` - Show this help message

Detach with `Ctrl+\` to leave the session running in the background.

**Examples:**
```bash
sprite console
sprite console -o myorg -s mysprite
```

## Checkpoints

### `sprite checkpoint create`

Create a new checkpoint.

```bash
sprite checkpoint create [flags]
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `--comment <text>` - Optional comment describing this checkpoint
- `-h, --help` - Show this help message

### `sprite checkpoint list`

List all checkpoints.

```bash
sprite checkpoint list [flags]
```

**Aliases:** `sprite checkpoint ls`, `sprite checkpoints ls`

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `--history <version>` - Filter by history version
- `--include-auto` - Include auto-generated checkpoints
- `-h, --help` - Show this help message

### `sprite checkpoint info`

Show information about a specific checkpoint.

```bash
sprite checkpoint info [flags] <version-id>
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite checkpoint info v2
```

### `sprite checkpoint delete`

Delete a checkpoint.

```bash
sprite checkpoint delete [flags] <version-id>
```

**Aliases:** `sprite checkpoint rm`

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite checkpoint delete v3
```

### `sprite restore`

Restore from a checkpoint version.

```bash
sprite restore [flags] <version-id>
```

**Aliases:** `sprite checkpoint restore`

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite restore v1
sprite restore -o myorg -s mysprite v2
```

## Networking

### `sprite proxy`

Forward local ports through the remote server proxy.

```bash
sprite proxy [flags] <port1> [port2] ...
sprite proxy [flags] <local1:remote1> [local2:remote2] ...
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-W, --stdio <[host]:port>` - Forward stdin/stdout to host:port on the sprite
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite proxy 8080
sprite proxy 3000 8080
sprite proxy 4005:4000
sprite proxy 3001:3000 8081:8080
sprite proxy -W :22
sprite proxy -o myorg -s mysprite 8080
```

### `sprite url`

Show or update sprite URL settings.

```bash
sprite url [flags]
sprite url update [flags]
```

URL format: `https://<sprite-name>-<org>.sprites.dev/`

Authentication modes:
- `sprite` - Org members only, plus org tokens (default)
- `public` - No authentication

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite url                         # Show current URL and auth setting
sprite url update --auth public    # Make URL publicly accessible
sprite url update --auth sprite    # Require org membership (default)
sprite url -o myorg -s mysprite    # Show URL for specific sprite
```

### `sprite url update`

Update URL authentication settings.

```bash
sprite url update --auth <type> [flags]
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-a, --auth <type>` - Authentication type: `public` or `sprite`
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite url update --auth public
sprite url update --auth sprite
sprite url update --auth public -s demo
```

## Utility Commands

### `sprite api`

Make authenticated API calls with curl.

```bash
sprite api [flags] <path> [curl options]
```

**Options:**
- `-o, --org <name>` - Specify organization
- `-s, --sprite <name>` - Specify sprite
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite api -o myorg /sprites
sprite api -o myorg -s my-sprite /upgrade -X POST
sprite api -o myorg -s my-sprite /exec -X GET
sprite api -o myorg -s my-sprite /checkpoints
```

### `sprite upgrade`

Upgrade the sprite client to the latest version.

```bash
sprite upgrade [flags]
```

**Options:**
- `--check` - Check for updates without installing
- `--force` - Force upgrade even if already up to date
- `--version <version>` - Upgrade to a specific version
- `--channel <channel>` - Release channel: `release`, `rc`, `dev`
- `-h, --help` - Show this help message

**Examples:**
```bash
sprite upgrade
sprite upgrade --check
sprite upgrade --force
sprite upgrade --channel dev
sprite upgrade --channel rc
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Command not found |
| 126 | Command cannot execute |
| 127 | Command not found (in sprite) |
| 128+ | Command terminated by signal |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SPRITE_TOKEN` | API token override (legacy; falls back if no stored token) |
| `SPRITE_URL` | Direct sprite URL (for local/dev direct connections) |
| `SPRITES_API_URL` | API URL override (default: `https://api.sprites.dev`) |

## Configuration Files

### Global Config

`~/.sprites/sprites.json` (managed by the CLI):
```json
{
  "version": "1",
  "current_selection": {
    "url": "https://api.sprites.dev",
    "org": "personal"
  },
  "urls": {
    "https://api.sprites.dev": {
      "url": "https://api.sprites.dev",
      "orgs": {
        "personal": {
          "name": "personal",
          "keyring_key": "sprites-cli:<user-id>",
          "use_keyring": true,
          "sprites": {}
        }
      }
    }
  }
}
```

### Local Context

`.sprite` (in project directory):
```json
{
  "organization": "personal",
  "sprite": "my-project-sprite"
}
```
