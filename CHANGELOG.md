# Changelog

All notable changes to **Breez** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.1.0] - 2026-08-01

### Added
- **Core Framing Protocol**: WebSocket payload framing supporting `TunnelInit`, `TunnelReady`, HTTP `Request`, HTTP `Response`, and Control frames (`internal/protocol`).
- **Gateway Server**: HTTP & WebSocket proxy server allocating subdomains and multiplexing traffic (`cmd/gateway`, `internal/gateway`).
- **Breez CLI**: Developer CLI supporting `breez serve <port>` and `breez version` (`cmd/breez`, `internal/cli`).
- **Installation Scripts**:
  - Linux/macOS bash installation & uninstallation scripts (`scripts/install.sh`, `scripts/uninstall.sh`).
  - Windows PowerShell installation & uninstallation scripts (`scripts/install.ps1`, `scripts/uninstall.ps1`).
- **Automated CI/CD & Auto-Releasing**:
  - GitHub Actions CI workflow (`.github/workflows/ci.yml`).
  - Automated tagging (`anothrNick/github-tag-action`) & GoReleaser release workflow (`.github/workflows/release.yml`, `.goreleaser.yaml`) for automatic version increments on pushes to `main`.
- **Documentation**: Project README (`README.md`) and initial architecture spec (`initial.md`).
- **Testing**: End-to-end integration tests verifying request forwarding from Gateway to CLI to local target server.
