# SSH Riders

SSH Riders is a multiplayer terminal game inspired by Tron light cycles. Players connect through an SSH-style gateway, join a shared room, and control riders that move continuously on a grid arena while leaving deadly trails behind.

## Architecture

- **Gateway**: session entrypoint, lobby UX, room join flow, terminal rendering bridge.
- **Room server**: authoritative tick loop, collision detection, round lifecycle, frame generation.
- **Orchestrator**: room registry, manifest loading, room lifecycle hooks.
- **Manifests**: YAML room definitions enabling room-per-container startup.

## Requirements

- Go 1.23+
- Docker / Docker Compose

## Quick start

```bash
make build
make run-orchestrator
make run-room
make run-gateway
```

Or with Docker Compose:

```bash
docker compose up --build
```

## Controls

- `w` — up
- `a` — left
- `s` — down
- `d` — right
- `q` — exit local round view

> In local terminal mode movement keys are read in raw mode, so **pressing Enter is not required** during the match.

## Repo layout

- `cmd/gateway` — gateway entrypoint
- `cmd/room` — room server entrypoint
- `cmd/orchestrator` — orchestrator entrypoint
- `internal/game` — authoritative game loop
- `internal/render` — ASCII frame builder
- `manifests` — room manifests
- `deployments` — container assets

## Roadmap

1. Replace local terminal session adapter with a real SSH server (`wish` or `gliderlabs/ssh`) when dependency fetch is available.
2. Add room auto-spawn and heartbeat-driven cleanup.
3. Add ANSI colors, spectator mode, and replay logging.
