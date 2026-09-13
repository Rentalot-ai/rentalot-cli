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

## API setup and trial preview

Configure the server origin, not `/api/v1`, and use the same origin for account sign-in, API-key issuance, CLI requests, and CRM read-back:

```bash
export RENTALOT_API_KEY="ra_..."
export RENTALOT_BASE_URL="http://localhost:3000" # local development trial
rentalot config show
```

Free Trial API access is development-only behind the server-side `trialApiAccess` gate. It permits account-owned, private, image-free property and contact CRUD. Production trial access remains disabled, and Starter remains read-only. Trial keys expire at the fixed account trial deadline and owner revocation is immediate. The dashboard keeps key listing and revocation available after expiry; only new trial-key issuance is gated.

Trial limits are shared by the account across active keys and key rotation: 20 requests/minute, 10 reads/minute and 5 writes/minute per resource, 200 requests/day, 700 requests over the trial, 20 resource writes/day, 50 resource writes over the trial, and 20 items per list page. Rotating keys does not reset those limits.

## Canonical property and contact writes

Property create requires `--address`, `--monthly-rent`, `--bedrooms`, and `--bathrooms`. Optional canonical fields include `--title`, `--property-type`, `--status`, `--is-public`, `--city`, `--state`, `--zip`, and `--description`. Canonical property types are `house`, `apartment`, `condo`, `townhouse`, and `room`; a studio is represented by `--bedrooms 0`. For the development-only trial first job, omit `--property-type` because that field is separately feature-gated, and keep the write private.

```bash
rentalot properties create \
  --title "Downtown Loft" --address "123 Main St" \
  --monthly-rent 1800 --bedrooms 0 --bathrooms 1 \
  --is-public=false
rentalot properties update "$PROPERTY_ID" --title "Downtown Loft Updated" --monthly-rent 1900
rentalot contacts create --name "Ada Lovelace" --email ada@example.com --role tenant
rentalot contacts update "$CONTACT_ID" --role tenant
```

Property lists support repeated canonical type filters and the exact Studio filter from the v1 API:

```bash
rentalot properties list --property-type apartment --property-type condo
rentalot properties list --bedroom-type studio --filter status=active
```

The repeated `--property-type` flags become repeated `propertyType` query parameters. `--bedroom-type studio` becomes `bedroomType=studio`; Studio means `bedrooms=0`, not a property type. The generic `--filter key=value` flag remains available alongside these property-specific filters.

Older `--name`, `--type`, and `--rent` property flags, plus contact `--type`, are deprecated aliases. They are translated to canonical JSON keys: property `name` becomes `title`, property `type` becomes `propertyType`, property `rent` becomes `monthlyRent`, and contact `type` becomes `role`. Contact `name` remains a canonical field. Contact creation requires a name and at least one of email or phone.

Use `--json` to preserve the API response envelope. A `401` means the key is missing, expired, or revoked, so configure an active key at the same origin. A `429` with `Retry-After` is a timed server instruction; the CLI reports it and does not retry automatically because attempts consume quota. A trial lifetime exhaustion response has no timed retry; upgrade to continue, and do not rotate keys to bypass account-wide limits.

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
