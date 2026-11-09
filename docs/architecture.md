# Architecture

DOKI uses a layered architecture so the terminal UI can be replaced later
without rewriting Docker operations.

## Layers

1. **Presentation (`internal/tui`)** — Bubble Tea models, Lip Gloss themes, screens.
2. **Application (`internal/app`)** — Use-cases such as listing/filtering containers.
3. **Domain (`internal/domain`)** — Models and ports (interfaces).
4. **Infrastructure (`internal/infrastructure`)** — Docker Engine SDK adapter.

## Rules

- TUI code must not call the Docker SDK directly.
- Application services depend on ports, not concrete clients.
- Domain models are independent of Bubble Tea and Docker API types.
- Destructive operations require explicit user confirmation in the TUI.

## Milestone 1 scope

- Docker ping/info
- Container list/inspect
- Start/stop/restart/pause/unpause/remove
- Themed responsive TUI
- Doctor command
