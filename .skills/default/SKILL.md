---
name: rentalot-cli
description: >
  CLI tool and Go library for the Rentalot v1 REST API — manage rental properties, contacts, showings, workflows, and more.
  Includes full API reference, schemas, bulk import, and product documentation.
license: MIT
compatibility:
  - Claude Code
  - Cursor
  - Codex
  - Gemini CLI
  - VS Code
metadata:
  author: Ariel Frischer
  version: 0.1.0
  tags: go, cli, library, rentalot, rental, api
allowed-tools: Bash Read Write Edit
---

# rentalot-cli

Go CLI and library for the Rentalot v1 REST API — manage rental properties, contacts, showings, conversations, workflows, webhooks, sessions, settings, and more.

> **Reference files** — this skill uses a references pattern. The main SKILL.md covers CLI usage, architecture, and development. Domain knowledge and schemas are in separate files — read them on-demand when you need the detail:
>
> - [api-schemas.md](api-schemas.md) — All OpenAPI component schemas: field types, enums, required fields. Read when writing Go structs, validation, or tests.
> - [product-docs.md](product-docs.md) — How Rentalot works: properties, photos, showings, workflows (step types, templates), agent behavior, slash commands, settings, FAQ. Read for domain context.
> - [bulk-import.md](bulk-import.md) — Bulk property import: CSV/Excel/JSON column schema, value coercion rules, auto-detected PMS exports (AppFolio, Buildium, Zillow, etc.), API bulk endpoint.
>
> **Note:** The CLI code in `cmd/rentalot/` and `pkg/rentalotcli/` defines the registered commands and transport. The Rentalot v1 OpenAPI document remains authoritative for request and response fields; registered resource coverage is not full API parity.
>
> **Live docs** (crawl for latest): https://rentalot.ai/llms.txt — index of all doc pages.
> - Product docs: `https://rentalot.ai/docs/{page}`
> - API reference: `https://rentalot.ai/docs/api-reference/{resource}`
> - OpenAPI spec: `https://rentalot.ai/api/v1/openapi.json`

## Config

```bash
export RENTALOT_API_KEY="ra_..."                   # API key (Settings > API Keys in dashboard)
export RENTALOT_BASE_URL="http://localhost:3000"   # default: https://rentalot.ai
```

The value is the server origin, not `/api/v1`. For a Free Trial, use a local development server with the server-side `trialApiAccess` gate enabled. Production keeps trial API access disabled. Create and verify the account, issue the key in **Settings > API Keys**, and use the same origin for sign-in, key issuance, CLI requests, and CRM read-back.

Free Trial access is limited to account-owned, private, image-free property and contact CRUD. Starter remains read-only. Trial limits are account-wide across all active keys and key rotation: 20 requests/minute, 10 reads/minute and 5 writes/minute per resource, 200 requests/day, 700 requests over the trial, 20 resource writes/day, 50 resource writes over the trial, and 20 items per list page. Trial keys expire at the fixed account trial deadline and revocation is immediate. Rotating a key does not reset quota.

## Commands

```bash
rentalot-cli --help                        # Show all commands
rentalot-cli version                       # Show version/build info
rentalot-cli completion bash               # Shell completion: bash|zsh|fish|powershell

# Properties
rentalot-cli properties list               # --property-type (repeatable) --bedroom-type studio --filter --limit --page
rentalot-cli properties get <id>
rentalot-cli properties create              # --address --monthly-rent --bedrooms --bathrooms
rentalot-cli properties update <id>         # --title --monthly-rent --property-type, etc.
rentalot-cli properties delete <id>
rentalot properties images presign <id>    # --file-name --content-type --size-bytes
rentalot properties images confirm <id>    # --r2-key --content-type --size-bytes [--alt-text]
rentalot properties images presign-batch <id> # --file presign.json ({"images": [...]})
rentalot properties images confirm-batch <id> # --file confirm.json ({"images": [...]})

# Contacts
rentalot-cli contacts list
rentalot-cli contacts get <id>
rentalot-cli contacts create               # --name plus --email or --phone; optional --role
rentalot-cli contacts update <id>          # --role, --status, or another canonical field
rentalot-cli contacts delete <id>

# Showings
rentalot-cli showings list
rentalot-cli showings get <id>
rentalot-cli showings create
rentalot-cli showings cancel <id>
rentalot-cli showings availability         # --property-id --date-from --date-to

# Conversations
rentalot-cli conversations list
rentalot-cli conversations get <id>
rentalot-cli conversations search          # --query

# Workflows
rentalot-cli workflows list
rentalot-cli workflows get <id>
rentalot-cli workflows create
rentalot-cli workflows update <id>
rentalot-cli workflows delete <id>

# Webhooks
rentalot-cli webhooks list
rentalot-cli webhooks get <id>
rentalot-cli webhooks create
rentalot-cli webhooks update <id>
rentalot-cli webhooks delete <id>
rentalot-cli webhooks test <id>

# Sessions
rentalot-cli sessions list
rentalot-cli sessions get <id>
rentalot-cli sessions review <id>          # --status approved|denied --notes

# Settings
rentalot-cli settings get                  # --json preserves nullable businessName
rentalot-cli settings update               # --business-name (max 100; empty clears)
rentalot-cli settings update --daily-leasing-brief-enabled=true --daily-leasing-brief-no-activity-enabled=false
```

Settings updates send `businessName` only when `--business-name` is supplied.
Use `rentalot-cli settings update --business-name ""` to clear the business name.
An omitted flag leaves it unchanged.

The `properties list`, `contacts list`, `showings list`, and `conversations list` commands accept `--output json` to preserve the full API response envelope and pagination metadata. They default to `--output table`. Other formats are rejected; global `--json` still selects JSON even with `--output table`.

Workflow availability is server-owned. Production v1 workflow, run, and session lists omit development-only lead qualification and its associated records; gated reads return `404 Not Found`, and triggering it is gated. OpenAPI shapes are unchanged. The workflow/session list commands accept empty lists and preserve the envelope with `--json`; their get commands return errors for `404` without successful resource output or automatic retries. A `404` can also mean an absent record, so check the selected environment and list available workflows instead of polling. There are no CLI workflow-run or workflow-execution subcommands; generic Go client callers must check HTTP status and decode API errors.

Image commands send upload metadata; upload the image bytes to the returned presigned URL before confirmation. Supported MIME types are `image/jpeg`, `image/png`, and `image/webp`, up to 10 MB. HEIC must be converted first. Confirm and confirm-batch screen the actual bytes for valid images and NSFW content; HTTP 422 means the rejected upload was deleted. The CLI preserves the server reason and reports that a suitable replacement needs a new presign/upload, with no automatic retry.

## Global Flags

```bash
--json      # Machine-readable JSON output
--limit N   # Max results per page
--page N    # Page number
--timeout 2m # Total HTTP request limit (default 30s; must be positive)
```

The timeout covers connection setup, redirects, and response-body reads for each request, not the whole multi-request command. A timeout does not trigger an automatic retry.

## Canonical first CRM job

Property writes use `title`, `address`, `monthlyRent`, `bedrooms`, `bathrooms`, and optional `propertyType`, `status`, and `isPublic`. `--name`, `--type`, and `--rent` remain deprecated flag aliases for older scripts, but the CLI translates them to property `title`, `propertyType`, and `monthlyRent`; those legacy property JSON keys are never sent. Contact `name` remains canonical, while contact `--type` is translated to `role`. Property types are `house`, `apartment`, `condo`, `townhouse`, and `room`. A studio is `--bedrooms 0`, not a property type. For the development-only trial first job, omit `propertyType` because it is separately feature-gated and keep `isPublic` false.

Property list filters use repeatable `--property-type` flags, which encode repeated singular `propertyType` query parameters. Use `--bedroom-type studio` for the exact v1 Studio filter, encoded as `bedroomType=studio`; Studio means `bedrooms=0`. The generic `--filter key=value` flag can be combined with these filters.

Property `--description` on create and update must not contain phone numbers, email addresses, or website links. Use the listing contact form for contact information. The server rejects contact info with HTTP `400`; the CLI preserves the validation message without retrying. Remove the contact info before resubmitting.

Contact writes use `role`, not `type`. A single contact create requires `--name` and at least one of `--email` or `--phone`; empty optional values are omitted. The role values are `prospect`, `tenant`, `landlord`, `property_manager`, `vendor`, and `other`.

```bash
rentalot properties create \
  --title "Downtown Loft" --address "123 Main St" \
  --monthly-rent 1800 --bedrooms 0 --bathrooms 1 \
  --is-public=false
rentalot contacts create \
  --name "Ada Lovelace" --email ada@example.com --role tenant
rentalot properties list \
  --property-type apartment --property-type condo
rentalot properties list --bedroom-type studio --filter status=active
```

Use `--json` for the canonical response envelope. The CLI does not create accounts or API keys. After a key expires or is revoked, issue or configure an appropriate active key at the same server origin. A `429` with `Retry-After` is a timed server instruction, but the CLI does not retry automatically because failed attempts consume quota. A `429` without a timed retry can indicate trial lifetime exhaustion; upgrade to continue rather than rotating keys.

For API requests that supply `Idempotency-Key`, the server atomically claims the key before execution. Concurrent requests or an unknown outcome return `409` with `Retry-After: 1` and same-key retry or reconciliation guidance. Wait at least one second, retain the same key and request when retrying, and reconcile an unknown outcome through resource or job read-back. Do not replace the key to bypass the conflict; doing so can duplicate the operation. Other `409` conflicts do not imply a timed retry. The CLI does not retry automatically and currently has no flag to supply `Idempotency-Key`.

## Development

```bash
make build          # Build binary -> bin/rentalot-cli
make test           # Run tests
make lint           # Run golangci-lint (falls back to go vet)
make format         # go fmt ./...
make run            # go run with version ldflags
make go-install     # Install to GOPATH/bin

make sync-schema    # Pull OpenAPI spec from running rentalot dev server
                    # Requires: make dev running in ../rentalot (http://localhost:3000)
                    # Writes to: internal/api/openapi.json
```

## Architecture

```
cmd/rentalot-cli/        # main.go -- cobra root, registers subcommands
internal/version/        # Version info (injected via ldflags at build time)
  internal/api/            # optional openapi.json -- synced from a running rentalot dev server
pkg/rentalotcli/         # Public library
  client.go              # HTTP client (Bearer auth, base URL, error unwrap)
  config.go              # Config loading from env
  properties.go          # Properties API methods
  contacts.go            # (etc per resource)
```

## Library Usage

```go
import "github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"

cfg := rentalotcli.ConfigFromEnv()
client := rentalotcli.NewClient(cfg)
props, err := client.ListProperties(ctx, rentalotcli.ListPropertiesParams{
    Status: "active",
    Limit:  50,
})
```

## Task Tracking

Project uses project-level beads (`.beads/`, gitignored, prefix `rentalot-cli-`):

```bash
bd ready                              # Open tasks, no blockers
bd query "label:needs-approval"       # Waiting for review
bd list                               # All open issues
```

## Sync Schema

```bash
# 1. Start rentalot dev server
cd ../rentalot && make dev

# 2. Pull latest OpenAPI spec
make sync-schema   # -> internal/api/openapi.json
```

When the rentalot API adds new routes, run `make sync-schema` then implement matching CLI commands.
Reference `../rentalot-mcp/src/tools/*.ts` for endpoint params and response shapes.

## Related Repos

- `../rentalot/` — main Next.js app + API server (`make dev` -> http://localhost:3000)
- `../rentalot-mcp/` — MCP server for same API (37 tools, TypeScript) — reference for endpoint params/shapes
