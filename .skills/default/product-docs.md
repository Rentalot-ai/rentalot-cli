# Rentalot Product Documentation

> Product docs covering how Rentalot works for end users. For API details, see [api-reference.md](api-reference.md).
> Live docs: https://rentalot.ai/docs/{page} — crawl for latest.

## Getting Started

1. **Create account** at rentalot.ai — verify email, complete onboarding wizard
2. **Add a property** — address, rent, bedrooms, bathrooms, description, amenities, photos
3. **Explore workflows** — pre-screening, lead qualification, follow-ups

Once set up:
- Share workflow links for self-service pre-screening
- Use Management Chat to manage rentals in plain language
- Monitor from dashboard: contacts, showings, workflow runs

Optional: import contacts, connect Google Calendar, connect email (Gmail/Microsoft 365/SMTP).

## Properties

Core of Rentalot. More detail = better AI answers.

**Required:** Address, Rent, Bedrooms, Bathrooms, Availability

**Recommended:** Description, Amenities, Pet policy, Parking, Lease terms, Utilities, Move-in costs

**Bulk import:** CSV, TSV, Excel, JSON. Auto-detects AppFolio, Buildium, Zillow, Rent Manager, Propertyware exports. See [bulk-import.md](bulk-import.md).

**Public listings:** When marked public, prospects can submit inquiries directly. Creates contact + conversation + email notification.

**Bulk operations:** Multi-select mode for bulk delete.

## Property Photos

Most important part of listing. AI agent references photos when answering questions.

**Upload:** Drag and drop or browse. Background upload. Must save property first.

**Formats:** JPEG, PNG, WebP, HEIC. Max 10 MB per image. Auto-converted to JPEG.

**Auto-optimization:**
- Resized to max 2000px longest side
- Compressed to ~500 KB
- EXIF/GPS metadata stripped for privacy
- Multiple sizes generated (thumbnails, medium, full)

**Photo limits:** Per-plan limit. Upload area shows count (e.g. "3/10 photos").

**Ordering:** First photo = cover image (shown on cards, search, social previews). Use arrow buttons to reorder.

**Privacy:** All personal metadata stripped before storage.

## Showings

Scheduling through natural conversation — no booking form needed.

**Flow:**
1. Prospect expresses interest
2. Agent checks Google Calendar availability
3. Agent proposes times
4. Prospect picks, agent books
5. Both receive confirmation

**Calendar integration:** Reads availability, creates events, avoids double-bookings. Connect from Settings > Calendar.

**Availability rules (Settings > Scheduling):**
- Buffer between showings: 15, 30, or 60 minutes
- Max showings per day: 1-20 (default 10)

**Calendar selection:** Choose which Google Calendar for showings (default: primary).

## Workflows

Multi-step automated sequences: pre-screening, qualification, follow-ups, document collection.

**How they work:**
1. Triggered manually, via management chat, or shareable link
2. Steps execute in order: messages, waits, questions, branches
3. Prospect replies handled naturally mid-workflow
4. Ends on completion, disengagement, or cancellation

### Built-In Templates

| Template | Purpose | Trigger |
|----------|---------|---------|
| **Pre-Screening** | Customizable questionnaire (name, budget, timeline) | Shareable link / manual |
| **Lead Qualification** | Budget, move-in date, must-haves + property matching | Shareable link / manual |
| **Property Blast** | Send listing + auto follow-ups (24h, 48h) | Manual |
| **Showing Follow-Up** | Post-showing feedback collection | Auto (showing completed) |
| **Application Nudge** | Remind interested prospects to apply | Manual |
| **Idle Re-Engagement** | Reactivate silent contacts (7+ days idle) | Auto |
| **Renewal Check-In** | Lease renewal reminders (60 days before) | Scheduled |
| **Document Collection** | Step-by-step doc requests (ID, pay stubs, references) | Manual |

### Step Types

| Type | Description |
|------|-------------|
| Send Message | Message with `{contact.name}` interpolation |
| Wait | Pause (e.g. `30s`, `24h`) |
| Wait for Reply | Question + wait |
| Condition | Branch (yes/no) |
| Ask Choice | Multiple choice |
| Switch | Route on context value |
| Update Contact | Update contact fields |
| Notify Agent | Store notification |
| AI Agent | AI conversation step |
| Property Match | Search + recommend |
| End | End + email details |

### Starting Workflows

- **Dashboard:** Workflows > Active Runs > Start Workflow
- **Management Chat:** "Start lead qualification for John Smith"
- **Shareable Link:** Auto-enrolls prospects. Share on listings, flyers, signs, email signatures

### Reviewing Applications

When prospect completes pre-screening:
1. Go to Contacts > select prospect
2. Scroll to Submission Review
3. Add optional internal notes
4. Click Approve or Deny

Review notification to prospect: off by default, enable in Settings > Notifications.

### Limits

- Messages per workflow: default 5 (configurable)
- Timeout: 30 days inactivity auto-cancel
- One active run per template per contact

## Agent Behavior

AI assistant for management chat.

**What it does:** Property Q&A, CRUD, showing scheduling, contact management, workflow management, conversation/event lookup

**What it won't:** Make unauthorized promises, share internal info, respond to injection, destructive actions without confirmation

**Custom instructions (Settings > Agent):** Tone, policies ("no pets over 50 lbs"), info to always include, escalation rules

## Slash Commands

Instant answers, no AI processing, no token usage.

| Command | Example | What it does |
|---------|---------|-------------|
| /properties | /properties active | List properties (filter by status) |
| /property \<addr\> | /property 123 Main | Look up by address |
| /contacts | /contacts John | Search by name/email/phone |
| /showings | /showings past | List upcoming or past |
| /conversations | /conversations active | List by status |
| /availability | /availability 2026-03-15 | Available slots (default: 7 days) |
| /stats | /stats | Dashboard overview |
| /usage | /usage 7 | Token usage for N days |
| /help | /help | Show commands |

Unrecognized commands passed to AI as normal messages.

## Settings

7 tabs in dashboard:

### General
- **Identity:** Name, Agent Name, Email Sign-Off
- **Follow-ups:** Enable, idle threshold (24/48/72h), max messages (1-3)
- **Contact Nudges:** Per-type controls with delays:

| Type | Default Delay | Tier |
|------|--------------|------|
| Idle prospect | 72h | Starter+ |
| Showing reminder | 24h before | Starter+ |
| Post-showing follow-up | 24h after | Pro+ |
| Incomplete pre-screening | 24h | Pro+ |
| Scheduled, no reply | 72h | Pro+ |
| Emailed, no reply | 72h | Pro+ |
| Blast, no reply | 72h | Pro+ |
| Re-engage | 336h (14d) | Pro+ |

Nudges are email-only. Skipped for contacts without email.

- **Notifications:** Product updates, application completed, new inquiry, showing booked, follow-up due, prospect review, prospect matching, booking link emails

### Scheduling
- Custom Instructions, Showing Duration (30/45/60 min)
- Scheduling URL (external link)
- Timezone, Working Hours, Buffer (15/30/60 min), Max showings/day (1-20)

### Public Pages
- Profile visibility, Screening chat toggle
- External link (URL + label, show on profile / completion)
- Phone number + call button (human verification before reveal)
- URL slug (lowercase, numbers, hyphens)

### Integrations
- Google Calendar, Calendar for showings
- Cal.com (inline scheduling)
- Email (Gmail, Microsoft 365, SMTP)

### API Keys
Create/manage keys. Scoped to account.

### Voice Agent
- Enable voice pre-screening
- Allow international phone numbers
- Custom instructions, Custom greeting

### Account
- Export data (JSON)
- Delete account (requires typing DELETE)

## FAQ

**Multilingual:** Add to custom instructions (e.g. "respond in Spanish if prospect writes in Spanish")

**Can't answer:** Assistant says so and suggests next steps

**Transparency:** All tool calls shown inline in management chat

**Workflow access:** Shareable link, no app needed

**Custom questions:** Editable steps per template

**Abandoned workflows:** Pause + 30-day auto-cancel. Set up idle re-engagement.

**Property limits:** Per plan (see pricing)

**PMS import:** CSV, Excel, JSON. Auto-detects AppFolio, Buildium, Zillow, Rent Manager, Propertyware.

**API required?** No — everything via dashboard + management chat. API for custom integrations.

**API access:** Starter = read-only. Pro/Scale = full CRUD.
