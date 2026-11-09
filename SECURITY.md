# Security Policy

## Scope

DOKI is a privileged infrastructure tool. Anyone who can talk to Docker Engine
through DOKI may effectively control containers on that host.

Supported versions: the default branch (`main`) and the latest published release.

## Reporting a vulnerability

Please report security issues **privately**:

1. Preferred: [GitHub Security Advisories](https://github.com/zpol/doki/security/advisories/new)
2. Or contact the repository owner (`@zpol`)

Include reproduction steps, impact, and affected versions when possible.
Please allow a reasonable window for triage before public disclosure.

## Product security commitments

- No network listener by default
- No telemetry
- No credential storage in DOKI config
- Confirmation for destructive actions
- Secret-safe defaults for environment display
- Terminal escape sanitization for log output
- Continuous checks: CI tests, `govulncheck`, CodeQL, OpenSSF Scorecard
- Known unfixed `github.com/docker/docker` AuthZ/plugin CVEs (daemon-side) are
  tracked in `.govulncheck-allowlist` until a fixed client release is available

## Security Inspector disclaimer

The built-in Security Inspector performs **configuration analysis** only.
It is not a vulnerability scanner and does not claim containers are “safe”
when no findings are reported.
