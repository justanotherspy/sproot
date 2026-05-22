# Sprite Services

## What Are Services?

Services are long-running processes managed by the sprite environment. Unlike regular commands started with `sprite exec`, services:

- **Persist across reboots** — automatically restart when the sprite boots
- **Keep the sprite alive** — a sprite won't pause while services are running
- **Auto-restart on crash** — exponential backoff from 1s to 60s, resets after 30s of stability
- **Support dependencies** — ordered startup with `--needs`

Use services for dev servers, databases, background workers, and anything that should always be running.

## Creating a Service

```bash
sprite-env services create <name> --cmd <command> [options]
```

### Create Options

| Option | Description |
|--------|-------------|
| `--cmd <command>` | Command to run (required) |
| `--args <arg1,arg2,...>` | Comma-separated arguments |
| `--env <KEY=val,...>` | Comma-separated environment variables |
| `--dir <path>` | Working directory |
| `--needs <svc1,svc2>` | Comma-separated service dependencies |
| `--http-port <port>` | HTTP port for proxy routing (only one service can have this) |
| `--duration <time>` | How long to stream logs after creation (default: 5s) |
| `--no-stream` | Don't stream logs after creation |

## CLI Reference

| Subcommand | Description |
|------------|-------------|
| `list` | List all services and their states |
| `get <name>` | Get a specific service's state |
| `create <name> [opts]` | Create a new service |
| `delete <name>` | Delete a service |
| `start <name>` | Start a stopped service |
| `stop <name>` | Stop a running service (sends TERM signal) |
| `restart <name>` | Restart a service (stop + start) |
| `signal <name> <signal>` | Send a signal to a service (e.g., TERM, HUP, KILL) |

Run `sprite-env services --help` for full usage.

## HTTP Port

Only one service can have `--http-port` configured. When set:

- The sprite's URL proxy routes incoming HTTP requests to that port (instead of the default 8080)
- The service auto-starts when a request arrives, even if it was stopped
- This is how dev servers become accessible via the sprite URL

If no service has `--http-port`, the proxy routes to port 8080 by default.

## Dependencies

Use `--needs` to declare startup dependencies between services:

```bash
sprite-env services create api --cmd node --args server.js --needs db
```

The `api` service will wait for `db` to start before launching. Dependency chains are resolved in order.

## Restart Behavior

Services automatically restart on crash:

- **Backoff**: starts at 1 second, doubles on each consecutive failure, caps at 60 seconds
- **Reset**: after 30 seconds of successful running, the backoff timer resets to 1s
- **Boot**: all services restart automatically when the sprite boots

## Lifecycle

- Services keep the sprite alive (it won't pause while services are running)
- Stopping all services and sessions allows the sprite to pause
- On boot, services start in dependency order
- Deleting a service stops it first

## Examples

### Dev server with HTTP port

```bash
sprite-env services create web --cmd node --args server.js --http-port 3000
```

The sprite URL will route HTTP traffic to port 3000. The service auto-starts on incoming requests.

### Background worker

```bash
sprite-env services create worker --cmd python --args worker.py --env QUEUE=default
```

Runs in the background, restarts on crash, persists across reboots.

### Service with dependencies

```bash
sprite-env services create db --cmd redis-server
sprite-env services create api --cmd node --args server.js --needs db --http-port 3000
```

The `api` service starts only after `db` is running.
