# sprites-artefacts

Reference snapshots of the context files a **fresh** sprite.dev sprite ships with
(captured before any `sproot setup` runs). Kept here so we can see the platform
baseline that `sproot` augments, without spinning up a sprite each time.

Captured from a `sprite create` sprite on 2026-05-22.
Sprite client/image version (`/.sprite/version.txt`): `0.0.1-rc43`.

| File | Source on sprite | Notes |
|------|------------------|-------|
| `llm.txt` | `/.sprite/llm.txt` | Platform LLM context. `sproot` appends a managed pointer block to this (see `internal/sprite/llmtxt.go`). |
| `llm-dev.txt` | `/.sprite/llm-dev.txt` | Dev-oriented platform LLM context. |
| `version.txt` | `/.sprite/version.txt` | Sprite image/client version string. |
| `docs/agent-context.md` | `/.sprite/docs/agent-context.md` | Platform agent-context guide. |
| `docs/docker.md` | `/.sprite/docs/docker.md` | Platform's own Docker-in-sprite guide (no systemd; `storage-driver: overlay2`; run dockerd as a sprite service). Informs the `docker` module's daemon.json handling. |
| `docs/services.md` | `/.sprite/docs/services.md` | Platform's sprite-env service guide. Informs the `sprite_service` module. |

These are a point-in-time reference, not consumed by any code path. Re-capture by
creating a sprite and reading the paths above (e.g. `sprite exec -s <name> -- cat /.sprite/llm.txt`).
