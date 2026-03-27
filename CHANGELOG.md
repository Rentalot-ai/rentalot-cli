# Changelog

All notable changes to rentalot-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Fixed

- bulk-import command now hits correct API endpoints (POST /api/v1/properties/bulk) and parses envelope response

### Changed

- bulk-import: removed --type flag (v1 API only supports property import)

## [0.1.0] - 2026-03-25

### Added

- version info from debug.ReadBuildInfo for go install users
- contacts command group — list, get, create, update, delete subcommands
- golangci-lint configuration with gocritic, misspell, and style checks
- govulncheck vulnerability scanning in CI pipeline
- go mod tidy check in CI to catch dependency drift
- CI triggers on dev branch for pre-merge validation
- properties command group — list, get, create, update, delete subcommands
- sessions command group — list, get, review subcommands
- webhooks command group — list, get, create, update, delete, test subcommands
- workflows command group — list, get, create, update, delete subcommands
- showings command group — list, get, create, update, cancel, check-availability subcommands
- conversations command group — list, get, search subcommands
- settings command group — get and update followup settings
- bulk-import command — import properties/contacts from CSV or JSON with job polling
- "--json" persistent flag for machine-readable output on all commands
- "--limit, --page, --filter" flags on all list commands
- table rendering for human-readable list output via text/tabwriter
- unit tests for API client — mock HTTP server, auth header injection, error decoding
- config management — load api_key + base_url from ~/.config/rentalot-cli/config.yaml with env var overrides
- API client package — thin HTTP wrapper with Bearer auth, base URL config, and RFC 9457 error decoding
- "config" command group — init, show, set, edit subcommands for managing global config

### Fixed

- CI lint failure — upgrade golangci-lint-action to v7 for golangci-lint v2 support
- CI govulncheck failure — bump Go to 1.25.8 to resolve stdlib vulnerabilities

## [0.0.1] - 2026-01-01

### Added

- Initial project scaffolding

[Unreleased]: https://github.com/Rentalot-ai/rentalot-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Rentalot-ai/rentalot-cli/compare/v0.0.1...v0.1.0
