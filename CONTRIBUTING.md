# Contributing to DOKI

Thanks for contributing. By participating you agree to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

1. Install Go matching `go.mod` (currently **1.27.3+**).
2. Ensure Docker Engine is available for manual testing.
3. Run:

```bash
make tidy
make test
make build
```

## Guidelines

- Keep Docker business logic out of the TUI layer.
- Prefer the official Docker SDK over shelling out.
- Do not fabricate metrics, security scores, or events.
- Add unit tests for domain/application logic.
- Destructive operations require confirmation UX.
- Keep PRs focused; prefer small, reviewable diffs.

## Pull requests

- Use the PR template.
- Include a short summary of what works now and how you tested it.
- Do not commit secrets, local Docker credentials, or `bin/` artifacts.
- CI must pass (`test`, `lint`, `govulncheck`).

## Security reports

Do not open public issues for vulnerabilities. Use
[GitHub Security Advisories](https://github.com/zpol/doki/security/advisories/new)
or see [SECURITY.md](SECURITY.md).
