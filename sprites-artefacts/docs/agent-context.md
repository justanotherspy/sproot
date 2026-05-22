# Sprite Environment

## Operating System
- Recent Ubuntu from Docker image
- No systemd, rc, or similar available, use Sprite services

## User Account
- Runs as `sprite` user
- Passwordless sudo access enabled

## HTTP Access

### URL Format
Each sprite has a unique URL: `https://<sprite-name>-<org>.sprites.dev/`

Example: `https://dev-mycompany.sprites.dev/`

### Routing
- By default, the proxy routes to port 8080
- Services can specify an `http_port` to receive proxied requests on a different port
- When an HTTP service is configured, it auto-starts on incoming requests

### Authentication Options

Sprite URLs support two authentication modes:

**1. Default (authenticated)**
- Requires Bearer token in Authorization header
- This is the secure default for development
- Access with:
  ```bash
  curl -H "Authorization: Bearer $SPRITE_API_TOKEN" https://mysprite-myorg.sprites.dev/
  ```

**2. Public**
- No authentication required - anyone with the URL can access
- Use for: public demos, webhooks, public APIs
- Enable with: `sprite url update --auth public`

### Managing URL Auth

```bash
# Check current URL and auth setting
sprite url

# Make URL public (no auth required)
sprite url update --auth public

# Restore authentication requirement
sprite url update --auth default
```

## Security

### CRITICAL: HTTP Services May Be Public
Sprite URLs can be configured for **public access**. Any HTTP service you create may become accessible to the entire internet. You MUST treat all HTTP endpoints as potentially public.

### Sensitive Information - NEVER Expose
LLM agents MUST NEVER:
- Create HTTP endpoints that expose environment variables, API keys, tokens, or credentials
- Serve file contents without explicit user request and proper access controls
- Create "debug", "admin", or "status" endpoints that dump system internals
- Log, print, or return secrets in HTTP responses or service output
- Build tools that proxy arbitrary file system access over HTTP
- Create endpoints that return unfiltered process info, user data, or system state

### Safe Patterns
- Implement authentication in your application if endpoints need protection
- Return only the minimum data necessary for functionality
- Validate and sanitize all inputs
- Use proper error handling that doesn't leak internal paths or stack traces
- Never hardcode secrets - use environment variables and never expose them via HTTP

## Lifecycle
- Sprites run while actively servicing HTTP requests
- Sprites run while detachable sessions are generating output
- When neither condition is true, sprites pause
- Sprites may remain paused or be fully stopped at any time
- When stopped, detachable sessions are lost
- Services restart automatically at boot time

## Services
- Long-running processes that persist between reboots
- Managed through the internal API via `sprite-env services` or the socket API
- Automatically restarted when sprite boots
- One service can have an `http_port` configured to receive proxied HTTP requests
- HTTP service auto-starts when proxy receives a request

## Checkpoints
- Point-in-time checkpoints and restores available
- Copy-on-write implementation for storage efficiency
- Last 5 checkpoints mounted at `/.sprite/checkpoints`
- Checkpoints capture only the writable overlay, not the base image

## File System
- Base Docker image with writable overlay
- Underlying image upgraded out-of-band without modifying overlay
- All modifications stored in overlay layer

## Internal API
- Unix socket at `/.sprite/api.sock`
- Manages environment, checkpoints, and services
- Run `sprite-env --help` for CLI documentation

## Development Tools
- If `llm-dev.txt` exists, it documents pre-installed language runtimes, version managers, and dev tools

## Network Policy

### Overview
Sprite environments enforce network egress policy through DNS-based filtering. Only allowed domains can be resolved and accessed.

### Viewing Current Policy
The active policy is read-only and mounted at:
```bash
cat /.sprite/policy/network.json
```

### Policy Format
Policies are JSON with a list of rules evaluated by specificity (exact match > subdomain wildcard > global wildcard):

```json
{
  "rules": [
    {"include": "defaults"},
    {"domain": "example.com", "action": "allow"},
    {"domain": "*.example.com", "action": "allow"},
    {"domain": "blocked.com", "action": "deny"}
  ]
}
```

**Include Rules**: `{"include": "defaults"}` loads common development domains (GitHub, npm, PyPI, Docker Hub, AI APIs like OpenAI/Claude/Copilot, etc.)

**Domain Rules**: Support exact matches, subdomain wildcards (`*.example.com`), and global wildcard (`*`)

**Empty Rules**: `{"rules": []}` means no enforcement (unrestricted mode)

### Updating Policy
**Network policies MUST be updated externally** - the container cannot modify its own policy.

Update via the Sprite API from outside the container:
```bash
curl -X POST https://api.sprites.dev/v1/sprites/{id}/policy/network \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"rules": [{"include": "defaults"}]}'
```

Policy changes reload automatically and terminate existing connections to newly-blocked domains.

### Behavior
- **Denied domains**: DNS returns REFUSED (fast-fail)
- **Raw IP connections**: Blocked unless the IP was resolved from an allowed domain
- **Private IPs**: Always blocked (except DNS to host)
- **Existing connections**: Terminated when policy changes via conntrack flush

### Troubleshooting
Test DNS resolution:
```bash
dig allowed-domain.com    # Should resolve
dig blocked-domain.com    # Returns REFUSED
```

