# Rentalot API Schemas

> All OpenAPI component schemas for the Rentalot v1 API.
> Source: https://rentalot.ai/api/v1/openapi.json

## ID Types

All IDs are `string (uuid)`:
- `PropertyId`, `ContactId`, `ConversationId`, `ShowingId`, `DraftId`, `FollowupId`
- `WorkflowId`, `WorkflowRunId`, `WebhookId`, `JobId`, `SessionId`

---

## Property

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (uuid) | Yes | Unique property identifier |
| userId | string | Yes | Owner tenant ID |
| address | string | Yes | Street address |
| city | string | Yes | City name |
| state | string | Yes | State/province code |
| zip | string | Yes | ZIP/postal code |
| monthlyRent | integer | Yes | Monthly rent in whole dollars |
| bedrooms | integer | Yes | Number of bedrooms |
| bathrooms | number | Yes | Number of bathrooms (supports 0.5 increments) |
| features | array[string] | Yes | List of property features/amenities |
| availabilityDate | string | Yes | ISO 8601 date (YYYY-MM-DD) |
| status | string | Yes | `active`, `rented`, `inactive`, `archived` |
| description | string | Yes | Free-text description |
| ownerId | string (uuid) | Yes | Optional owner contact reference |
| createdAt | string | Yes | ISO 8601 timestamp |
| updatedAt | string | Yes | ISO 8601 timestamp |

## CreatePropertyRequest

Required: `address`, `monthlyRent`, `bedrooms`, `bathrooms`

| Field | Type | Description |
|-------|------|-------------|
| address | string | Street address |
| city | string | City |
| state | string | State/province |
| zip | string | ZIP/postal code |
| monthlyRent | integer | Monthly rent (whole dollars) |
| bedrooms | integer | Bedrooms |
| bathrooms | number | Bathrooms (0.5 increments) |
| features | array[string] | Features/amenities |
| availabilityDate | string (date) | Available date |
| status | string | `active` / `rented` / `inactive` / `archived` |
| isPublic | boolean | Public listing visibility |
| description | string | Property description |
| ownerId | string (uuid) | Owner contact ID |
| amenities | array[string] | Amenity list |
| petPolicy | string | `allowed` / `not-allowed` / `negotiable` |
| parking | string | `included` / `available` / `none` |
| laundry | string | `in-unit` / `in-building` / `none` |
| leaseMinMonths | integer | Minimum lease length |
| leaseMaxMonths | integer | Maximum lease length |
| moveInDate | string (date) | Move-in date |
| depositAmount | integer | Security deposit (dollars) |
| utilitiesIncluded | array[string] | Included utilities |
| squareFootage | integer | Square footage |
| yearBuilt | integer | Year built |
| neighborhoodDescription | string | Neighborhood info |
| url | string (uri) | External listing URL |
| internalNotes | string | Internal notes (not shown to prospects) |

## UpdatePropertyRequest

Same fields as `CreatePropertyRequest`, all optional. Send only fields to change.

---

## Contact

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string (uuid) | Yes | |
| name | string | Yes | Full name |
| email | string | Yes | Email address |
| phone | string | Yes | Phone (E.164) |
| status | string | Yes | `prospect` / `scheduled` / `applicant` / `renter` / `archived` |
| channelPreference | string | Yes | Preferred messaging channel |
| source | string | Yes | Acquisition source |
| referralSource | string | Yes | External referral |
| createdAt | string | Yes | ISO 8601 |
| updatedAt | string | Yes | ISO 8601 |

## CreateContact

Required: `name` + at least one of `email` or `phone`

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| name | string | -- | Contact name |
| email | string (email) | -- | Email |
| phone | string | -- | Phone |
| status | string | prospect | `prospect` / `scheduled` / `applicant` / `renter` / `archived` |
| role | string | prospect | `prospect` / `tenant` / `landlord` / `property_manager` / `vendor` / `other` |
| channelPreference | string | -- | Preferred channel |
| source | string | api | Acquisition source |
| referralSource | string | -- | Referral source |
| language | string | en | ISO 639-1 code |

## UpdateContact

All optional: `name`, `email`, `phone`, `status`, `role`, `channelPreference`, `source`, `referralSource`, `language`, `notes`

---

## Conversation

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| contactId | string (uuid) | Contact ID |
| channel | string | whatsapp, telegram, sms, gmail |
| status | string | `active` / `archived` |
| lastMessageAt | string | ISO 8601 |
| messageCount | integer | Total messages |
| createdAt | string | ISO 8601 |

## ConversationMessage

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| conversationId | string (uuid) | |
| direction | string | `inbound` / `outbound` |
| content | string | Message text |
| channel | string | Channel |
| sentAt | string | ISO 8601 |

## ConversationSearchResult

| Field | Type | Description |
|-------|------|-------------|
| messageId | string (uuid) | |
| content | string | Message content |
| direction | string | `inbound` / `outbound` |
| channel | string | |
| sentAt | string | ISO 8601 |
| conversationId | string (uuid) | |
| contactName | string | Contact name |

---

## Showing

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| propertyId | string (uuid) | Property |
| contactId | string (uuid) | Contact |
| title | string | Showing title |
| description | string | Details |
| startTime | string | ISO 8601 |
| endTime | string | ISO 8601 |
| timeZone | string | IANA timezone |
| location | string | Address |
| status | string | `pending` / `confirmed` / `completed` / `cancelled` |
| notes | string | Internal notes |
| source | string | Creation source |
| type | string | `showing` |
| createdAt | string | ISO 8601 |
| updatedAt | string | ISO 8601 |

## CreateShowing

Required: `propertyId`, `contactId`, `title`, `startTime`, `endTime`

Optional: `description`, `timeZone`, `location`, `notes`

## UpdateShowing

All optional: `title`, `description`, `startTime`, `endTime`, `timeZone`, `location`, `status` (`pending`/`confirmed`/`cancelled`/`completed`), `notes`

## AvailabilitySlot

| Field | Type | Description |
|-------|------|-------------|
| date | string | YYYY-MM-DD |
| start | string | ISO 8601 |
| end | string | ISO 8601 |

---

## Event

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| propertyId | string (uuid) | |
| contactId | string (uuid) | |
| title | string | |
| description | string | |
| startTime | string | ISO 8601 |
| endTime | string | ISO 8601 |
| timeZone | string | IANA |
| location | string | |
| status | string | `pending` / `confirmed` / `completed` / `cancelled` |
| notes | string | |
| type | string | `showing` / `call` / `inspection` / `meeting` |
| source | string | `internal` / `google_calendar` / `calcom` |
| createdAt | string | ISO 8601 |
| updatedAt | string | ISO 8601 |

---

## SendMessage

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| contactId | string (uuid) | Yes | |
| body | string | Yes | Message text |
| channel | string | No | `sms` / `whatsapp` / `email` / `gmail` / `telegram` |

## SentMessage

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| content | string | Message body |
| channel | string | `sms` / `whatsapp` / `email` / `gmail` / `telegram` |
| direction | string | `outbound` |
| sentAt | string | ISO 8601 |

---

## Draft

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| contactId | string (uuid) | Target contact |
| propertyId | string (uuid) | Optional property context |
| content | string | Draft body |
| channel | string | `sms` / `whatsapp` / `email` / `gmail` / `telegram` |
| status | string | `pending` / `sent` / `expired` |
| expiresAt | string | ISO 8601 (24h after creation) |
| createdAt | string | ISO 8601 |
| updatedAt | string | ISO 8601 |

## CreateDraft

Required: `contactId`, `channel`, `body`

Optional: `recipientPhone`, `recipientEmail`

## UpdateDraft

Optional: `body`, `channel`, `recipientPhone`, `recipientEmail`

---

## Followup

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| contactId | string (uuid) | |
| conversationId | string (uuid) | |
| message | string | Follow-up content |
| scheduledFor | string | ISO 8601 |
| status | string | `pending` / `processing` / `sent` / `cancelled` / `failed` |
| createdAt | string | ISO 8601 |

## CreateFollowup

Required: `contactId`, `conversationId`, `scheduledAt`

Optional: `sequenceStep` (integer)

---

## WorkflowTemplate (List)

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| name | string | Display name |
| slug | string | URL-safe slug |
| description | string | Description |
| isActive | boolean | Accepts new runs |
| isPublic | boolean | Public chat access |
| createdAt | string | ISO 8601 |

## WorkflowTemplateFull (Detail)

Extends WorkflowTemplate with:

| Field | Type | Description |
|-------|------|-------------|
| steps | array[object] | Ordered step definitions |
| triggerType | string | `manual` / `deep_link` / `automatic` / `scheduled` |
| triggerConfig | object | Trigger-specific config |
| exitConditions | object | `{ maxMessages, timeoutDays }` |
| questionConfig | object | OQAAT form config |
| completionConfig | object | Completion screen config |
| introMessage | string | Opening message |
| voiceConfig | object | Voice agent config |
| currentVersion | number | Template version |
| updatedAt | string | ISO 8601 |

### Step Types

| Type | Description |
|------|-------------|
| send_message | Send a message (supports `{contact.name}` interpolation) |
| wait | Pause for a duration (e.g. `30s`, `24h`) |
| wait_for_reply | Ask a question and wait |
| condition | Branch based on condition (yes/no) |
| ask_choice | Multiple-choice options |
| switch | Route based on context value |
| update_contact | Update contact fields |
| notify_agent | Store notification summary |
| agent | AI-powered conversation step |
| property_match | Search listings and recommend |
| end | End workflow and email details |

## CreateWorkflowTemplate

Required: `name`, `steps`, `triggerType`

Optional: `description`, `triggerConfig`, `exitConditions`, `questionConfig`, `completionConfig`, `isPublic`, `introMessage`, `voiceConfig`, `isActive`

## UpdateWorkflowTemplate

All fields from Create, all optional. Version auto-increments on execution field changes.

## WorkflowRun

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| templateId | string (uuid) | Source template |
| contactId | string (uuid) | |
| propertyId | string (uuid) | Optional property context |
| status | string | `pending` / `running` / `paused` / `completed` / `cancelled` / `failed` |
| currentStepIndex | number | Zero-based step index |
| messagesSent | number | Total messages sent |
| exitReason | string | Why run ended (null if active) |
| startedAt | string | ISO 8601 |
| completedAt | string | ISO 8601 (null if active) |

## TriggerWorkflow

Required: `workflowId`, `contactId`

Optional: `propertyId`

---

## WebhookSubscription

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| url | string (uri) | HTTPS endpoint |
| events | array[string] | Subscribed event types |
| active | boolean | Delivering events |
| description | string | Label |
| lastDeliveredAt | string | ISO 8601 |
| lastFailedAt | string | ISO 8601 |
| createdAt | string | ISO 8601 |
| updatedAt | string | ISO 8601 |

## CreateWebhook

Required: `url` (HTTPS), `events`

Optional: `description`

## UpdateWebhook

All optional: `url`, `events`, `active`, `description`

---

## PropertyImage

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| url | string (uri) | CDN URL (Cloudflare Image Transformations) |
| altText | string | Alt text |
| order | integer | Display order (0-indexed) |

## V1PresignImageRequest

Required: `fileName`, `contentType` (`image/jpeg`/`image/png`/`image/webp`/`image/heic`), `sizeBytes` (max 10MB)

## V1ConfirmImageRequest

Required: `r2Key`, `contentType`, `sizeBytes`

Optional: `altText`

## V1PresignImageBatchRequest

`{ "images": [{ "fileName", "contentType", "sizeBytes" }] }` (max 20)

## V1ConfirmImageBatchRequest

`{ "images": [{ "r2Key", "contentType", "sizeBytes", "altText?" }] }`

## ImportImagesRequest

`{ "urls": ["https://..."] }`

## BulkCreatePropertiesRequest

`{ "properties": [{ ...flexible field names... }] }` (max 500)

---

## Session

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | |
| contactId | string (uuid) | |
| workflowTemplateId | string (uuid) | |
| submission | object | Key-value pairs from form |
| status | string | `active` / `completed` / `expired` / `draft` |
| reviewStatus | string | `pending_review` / `approved` / `denied` |
| reviewNotes | string | Internal notes |
| reviewedAt | string | ISO 8601 |
| createdAt | string | ISO 8601 |
| updatedAt | string | ISO 8601 |
| expiresAt | string | ISO 8601 |

## ReviewSession

Required: `reviewStatus` (`approved` / `denied`)

Optional: `reviewNotes` (max 2000 chars)

---

## AgentSettings

| Field | Type | Description |
|-------|------|-------------|
| agentName | string | AI agent display name |
| customInstructions | string | Custom system prompt |
| timezone | string | IANA timezone |
| workingHours | WorkingHours | `{ start, end, days }` |
| defaultShowingDuration | number | 30, 45, or 60 minutes |
| emailSignOff | string | Email signature |
| publicChatIntro | string | Public chat welcome |
| publicChatEnabled | boolean | |
| profilePublic | boolean | |
| licenseNumber | string | RE license |
| publicPhone | string | Public phone |
| followupEnabled | boolean | |
| followupIdleHours | number | 24, 48, or 72 |
| followupMaxSteps | number | 1-3 |
| voiceEnabled | boolean | |
| voicePromptAddition | string | Voice prompt extra |
| voiceFirstMessage | string | Voice greeting |
| allowInternationalPhone | boolean | |
| bookingUrl | string | External booking URL |
| externalLinkUrl | string | |
| externalLinkLabel | string | |
| showExternalLinkOnProfile | boolean | |
| showExternalLinkOnCompletion | boolean | |
| callHumanEnabled | boolean | |
| followupTemplates | array[object] | `[{ step: 1, template: "..." }]` — placeholders: `{contactName}`, `{agentName}`, `{propertyAddress}` |
| emailPreferences | EmailPreferences | |

## WorkingHours

| Field | Type | Description |
|-------|------|-------------|
| start | string | HH:mm (24h) |
| end | string | HH:mm (24h) |
| days | array[number] | ISO weekdays (1=Mon, 7=Sun) |

## EmailPreferences

| Field | Type | Description |
|-------|------|-------------|
| marketing | boolean | Marketing emails |
| applicationNotifications | boolean | Application notifications |
| notifyProspectOnReview | boolean | Email prospect on review |
| emailProspectMatches | boolean | Email matching properties |
| schedulingEmailEnabled | boolean | Scheduling confirmations |

## FollowupSettings

| Field | Type | Description |
|-------|------|-------------|
| enabled | boolean | Auto follow-ups on/off |
| idleHours | number | Hours before follow-up |
| maxSteps | number | Max sequence steps |

## ErrorResponse

```json
{
  "error": {
    "code": "string",
    "message": "string",
    "details": [{ "field": "string", "message": "string" }]
  }
}
```
