# DOKI

[![CI](https://github.com/zpol/doki/actions/workflows/ci.yml/badge.svg)](https://github.com/zpol/doki/actions/workflows/ci.yml)
[![CodeQL](https://github.com/zpol/doki/actions/workflows/codeql.yml/badge.svg)](https://github.com/zpol/doki/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/zpol/doki/badge)](https://securityscorecards.dev/viewer/?uri=github.com/zpol/doki)
[![Go Report Card](https://goreportcard.com/badge/github.com/zpol/doki)](https://goreportcard.com/report/github.com/zpol/doki)
[![License](https://img.shields.io/github/license/zpol/doki)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zpol/doki)](go.mod)
[![Release](https://img.shields.io/github/v/release/zpol/doki?include_prereleases&sort=semver)](https://github.com/zpol/doki/releases)
[![Issues](https://img.shields.io/github/issues/zpol/doki)](https://github.com/zpol/doki/issues)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

```text
◈ DOKI — Docker Operations Cockpit
Your Containers. One Terminal. Total Control.
```

**DOKI** is an open-source, terminal-native Docker operations cockpit written in Go.
It connects to a **real Docker Engine** and provides keyboard-driven container
management with a visually rich, K9s/btop-inspired interface.

> No fake data. No telemetry. No daemon. Just your Docker Engine in a serious TUI.

## Why DOKI

| Principle | What it means |
|---|---|
| Real Docker only | Official Engine SDK — never parses `docker ps` |
| Operator-first UX | Keyboard-driven workflows inspired by K9s |
| Honest metrics | Container-aggregated stats; no invented host gauges |
| Safe by default | Confirmations, secret redaction, no volume wipe on remove |
| Layered Go design | TUI → app → ports → Docker adapter |

## Status

Implemented through **Milestones 1–3** plus Compose label discovery:

| Area | Status |
|---|---|
| Real Docker SDK integration | Done |
| Themed Bubble Tea v2 TUI | Done |
| Containers list / inspect / lifecycle / bulk mark | Done |
| Live logs + interactive shell | Done |
| Stats / braille graphs / dashboard / events | Done |
| Images / volumes / networks inspect | Done |
| Command palette (`:`) | Done |
| Security Inspector | Done |
| Diagnostics engine | Done |
| Compose project discovery (labels) | Done |
| Multi-host contexts / demo mode / release automation | Planned |

## Features

- Official Docker Engine Go SDK
- Full-screen TUI: Bubble Tea v2 + Lip Gloss v2
- Themes: `cyberpunk` (default), `matrix`, `nord`, `monochrome` (+ `NO_COLOR`)
- Containers: filter, mark (`space`), inspect, start/stop/restart/pause/unpause/remove
- Live logs with follow/pause/search and ANSI sanitization
- Interactive shell via Docker exec (suspends TUI)
- Dashboard with braille graphs, top consumers, and Docker events
- Images, volumes, networks list + detail inspect
- Security configuration findings (privileged, socket mounts, caps, root, ports…)
- Diagnostics (unhealthy, restarts, OOM, nonzero exits)
- Compose projects discovered from container labels
- `doki doctor` environment checks

## Requirements

- Go **1.27+** (see `go.mod`)
- Docker Engine reachable locally or via `DOCKER_HOST`

## Install / Build

```bash
git clone https://github.com/zpol/doki.git
cd doki
make build
./bin/doki doctor
./bin/doki
```

## Usage

```bash
doki
doki --theme nord
doki --host unix:///var/run/docker.sock
doki doctor
doki --version
```

### Keyboard

| Key | Action |
|---|---|
| `j` / `k` | Navigate |
| `space` | Mark / unmark for bulk actions |
| `A` / `U` | Mark all / clear marks |
| `Enter` / `i` | Inspect |
| `l` | Logs |
| `S` | Shell |
| `d` | Diagnostics for selection |
| `D` | Remove (confirm) |
| `s` / `x` / `R` / `p` | Start / Stop / Restart / Pause |
| `/` | Filter |
| `:` | Command palette |
| `Tab` | Cycle views |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |

More detail: [docs/keyboard-shortcuts.md](docs/keyboard-shortcuts.md)

### Command palette

```text
:dashboard  :containers  :images  :volumes  :networks
:compose    :security    :diagnostics  :theme nord
:refresh    :help        :quit
```

## Configuration

`$XDG_CONFIG_HOME/doki/config.yaml` (often `~/.config/doki/config.yaml`)

```yaml
theme: cyberpunk
refresh:
  containers: 1s   # minimum 500ms
display:
  show_stopped: true
confirmations:
  destructive_actions: true
```

## Architecture

```text
TUI (Bubble Tea)
  → Application services (containers, logs, metrics, security, diagnostics…)
    → Domain ports
      → Docker Engine SDK adapter
```

See [docs/architecture.md](docs/architecture.md).

## Security & compliance

DOKI is a privileged infrastructure tool: access to Docker Engine can mean
host-level control. Design commitments:

- No network listener / no telemetry
- Destructive actions require confirmation
- Likely-secret environment values are redacted by default
- Container remove does **not** delete volumes
- Security Inspector is **configuration analysis**, not a vulnerability scanner
- CI: unit/race tests, `govulncheck`, golangci-lint, CodeQL, OpenSSF Scorecard

Report vulnerabilities via [GitHub Security Advisories](https://github.com/zpol/doki/security/advisories/new).
Policy: [SECURITY.md](SECURITY.md).

## Development

```bash
make fmt && make vet && make test && make build
```

Contribution guide: [CONTRIBUTING.md](CONTRIBUTING.md)  
Code of Conduct: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## License

Apache License 2.0 — see [LICENSE](LICENSE).
