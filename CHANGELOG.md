# Changelog

All notable changes to rentalot-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- contacts command group — list, get, create, update, delete subcommands
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

## [0.0.1] - 2026-01-01

### Added

- Initial project scaffolding

[Unreleased]: https://gitlab.com/ariel-frischer/rentalot-cli/-/compare/v0.0.1...HEAD
