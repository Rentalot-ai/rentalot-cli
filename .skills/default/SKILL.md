---
name: rentalot-cli
description: >
  CLI tool for managing Rentalot rental properties, contacts, and workflows
license: MIT
compatibility:
  - Claude Code
  - Cursor
  - Codex
  - Gemini CLI
  - VS Code
metadata:
  author: Ariel Frischer
  version: 0.0.1
  tags: go, cli, library
allowed-tools: Bash Read Write Edit
---

# rentalot-cli

CLI tool for managing Rentalot rental properties, contacts, and workflows

## Commands

```bash
rentalot-cli --help              # Show available commands
rentalot-cli version             # Show version info
rentalot-cli completion bash     # Shell completion: bash|zsh|fish|powershell
```

## Library Usage

```go
import "github.com/ariel-frischer/rentalot-cli/pkg/rentalotcli"
```

## Development

```bash
make build          # Build binary
make test           # Run tests
make lint           # Run linters
make format         # Format code
```
