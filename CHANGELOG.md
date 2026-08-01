# Changelog

All notable changes to **Breez** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Core Framing Protocol**: WebSocket payload framing supporting `TunnelInit`, `TunnelReady`, HTTP `Request`, HTTP `Response`, and Control frames (`internal/protocol`).
- **Gateway Server**: HTTP & WebSocket proxy server allocating subdomains and multiplexing traffic (`cmd/gateway`, `internal/gateway`).
- **Breez CLI**: Developer CLI supporting `breez serve <port>` and `breez version` (`cmd/breez`, `internal/cli`).
- **Installation Scripts**:
  - Linux/macOS bash installation & uninstallation scripts (`scripts/install.sh`, `scripts/uninstall.sh`).
  - Windows PowerShell installation & uninstallation scripts (`scripts/install.ps1`, `scripts/uninstall.ps1`).
- **CI/CD & Versioning**:
  - GitHub Actions CI workflow (`.github/workflows/ci.yml`).
  - GoReleaser release workflow (`.github/workflows/release.yml`, `.goreleaser.yaml`) supporting Linux, macOS, and Windows cross-compilation.
- **Documentation**: Project README (`README.md`) and initial architecture spec (`initial.md`).
- **Testing**: End-to-end integration tests verifying request forwarding from Gateway to CLI to local target server.
