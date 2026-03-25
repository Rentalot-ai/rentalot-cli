<div align="center">

**rentalot-cli**

CLI tool for managing Rentalot rental properties, contacts, and workflows
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

</div>

## Install

**Go install**:

```bash
go install github.com/Rentalot-ai/rentalot-cli/cmd/rentalot@latest
```

**Go get** (library):

```bash
go get github.com/Rentalot-ai/rentalot-cli
```

**From source**:

```bash
git clone https://github.com/Rentalot-ai/rentalot-cli.git
cd rentalot-cli
make build    # Binary at bin/rentalot
```

## Usage

```bash
rentalot --help
```

## Library Usage

```go
import "github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
```

### AI Agent Skill

This project ships a [SKILL.md](.skills/default/SKILL.md) following the [Agent Skills open standard](https://agentskills.io). Install it so your coding agent knows all commands and options.

**Quick install with [`skills`](https://skills.sh) CLI** (by Vercel Labs):

```bash
npx skills add Rentalot-ai/rentalot-cli
```

<details>
<summary><strong>Manual install</strong></summary>

**Claude Code** — Skills live in `~/.claude/skills/` (global) or `.claude/skills/` (project-local).

```bash
# Global — available in all projects
mkdir -p ~/.claude/skills/rentalot-cli
curl -fsSL https://raw.githubusercontent.com/Rentalot-ai/rentalot-cli/main/.skills/default/SKILL.md \
  -o ~/.claude/skills/rentalot-cli/SKILL.md

# Project-local — checked into this repo only
mkdir -p .claude/skills/rentalot-cli
curl -fsSL https://raw.githubusercontent.com/Rentalot-ai/rentalot-cli/main/.skills/default/SKILL.md \
  -o .claude/skills/rentalot-cli/SKILL.md
```

**Codex CLI** — reads skills from `~/.codex/skills/` (global) or `.codex/skills/` (project-local).

```bash
# Global
mkdir -p ~/.codex/skills/rentalot-cli
curl -fsSL https://raw.githubusercontent.com/Rentalot-ai/rentalot-cli/main/.skills/default/SKILL.md \
  -o ~/.codex/skills/rentalot-cli/SKILL.md

# Project-local
mkdir -p .codex/skills/rentalot-cli
curl -fsSL https://raw.githubusercontent.com/Rentalot-ai/rentalot-cli/main/.skills/default/SKILL.md \
  -o .codex/skills/rentalot-cli/SKILL.md
```

Or pass directly: `codex --instructions .skills/default/SKILL.md`

</details>

## Development

```bash
make build          # Build binary
make test           # Run tests
make lint           # Run linters
make format         # Format code
```

## Shell Completion

```bash
# Bash
source <(rentalot completion bash)

# Zsh
source <(rentalot completion zsh)

# Fish
rentalot completion fish > ~/.config/fish/completions/rentalot.fish
```

## License
[MIT](LICENSE)
