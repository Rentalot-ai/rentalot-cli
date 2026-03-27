# Bulk Property Import

> Import properties from spreadsheets (CSV, Excel, JSON) via dashboard or API.

## Dashboard Import

Go to Properties > Import, drag and drop or browse.

### Supported Formats

| Format | Extensions | Notes |
|--------|-----------|-------|
| CSV | .csv | Most common |
| TSV | .tsv | Tab-separated |
| Excel | .xlsx .xls | |
| JSON | .json | Array of objects |

**Limits:** 500 rows max, 10 MB max file size.

### Import Flow

1. **Upload** — file parsed in browser (nothing uploaded yet)
2. **Auto-detect** — recognized PMS exports get pre-mapped automatically
3. **Map columns** — review mapping, adjust or skip columns
4. **Validate** — check for errors, fix or skip invalid rows
5. **Import** — confirmed rows created as properties

### Auto-Detected PMS Exports

| Tool | Detected By |
|------|------------|
| AppFolio | Unit Directory report (merges split address columns) |
| Buildium | Unit list export (merges multi-line addresses) |
| Zillow | Property listing export |
| Rent Manager | Unit report |
| Propertyware | Property export (includes year built, pet policy, deposit) |

Shows badge like "Detected: AppFolio Unit Directory". All columns pre-mapped.

### Column Name Matching

For unrecognized files, generic column name matching:

| Property Field | Recognized Column Names |
|---------------|------------------------|
| Address | `address`, `street`, `street_address`, `location` |
| Monthly Rent | `rent`, `monthly_rent`, `price`, `rent_amount` |
| Bedrooms | `bedrooms`, `beds` |
| Bathrooms | `bathrooms`, `baths` |
| Square Footage | `sqft`, `square_footage`, `sq_ft`, `area` |
| Pet Policy | `pets`, `pet_policy`, `pet_friendly` |
| Deposit | `deposit`, `security_deposit` |

### Value Coercion

Values are automatically normalized:

| Input | Output |
|-------|--------|
| `"$1,500"` | 1500 |
| `"3 BR"` | 3 |
| `"Yes"` | allowed |
| `"03/15/2026"` | 2026-03-15 |

### Required Columns

Minimum: **Address**, **Rent**, **Bedrooms**, **Bathrooms**

The more fields included, the better the AI agent answers prospect questions.

## CSV Schema

Example CSV with all supported columns:

```csv
address,city,state,zip,rent,bedrooms,bathrooms,description,features,availability_date,status,pet_policy,parking,laundry,lease_min_months,lease_max_months,move_in_date,deposit,utilities_included,sqft,year_built,neighborhood_description,url,internal_notes
"123 Main St, Apt 4B",New York,NY,10001,2500,2,1,"Sunny 2BR with updated kitchen","hardwood floors,dishwasher,AC",2026-04-01,active,allowed,included,in-unit,12,24,2026-04-01,2500,"water,trash",950,1985,"Quiet tree-lined block near subway",https://example.com/listing,"Corner unit, gets afternoon sun"
```

### All Property Fields

| CSV Column | API Field | Type | Required | Values |
|-----------|-----------|------|----------|--------|
| address | address | string | Yes | Full street address |
| city | city | string | No | City name |
| state | state | string | No | State/province code |
| zip | zip | string | No | ZIP/postal code |
| rent / monthly_rent / price | monthlyRent | integer | Yes | Whole dollars |
| bedrooms / beds | bedrooms | integer | Yes | |
| bathrooms / baths | bathrooms | number | Yes | Supports 0.5 (e.g. 1.5) |
| description | description | string | No | Free text |
| features | features | string (comma-sep) | No | e.g. "hardwood,dishwasher" |
| availability_date | availabilityDate | date | No | YYYY-MM-DD |
| status | status | string | No | `active` / `rented` / `inactive` / `archived` |
| pets / pet_policy | petPolicy | string | No | `allowed` / `not-allowed` / `negotiable` |
| parking | parking | string | No | `included` / `available` / `none` |
| laundry | laundry | string | No | `in-unit` / `in-building` / `none` |
| lease_min_months | leaseMinMonths | integer | No | |
| lease_max_months | leaseMaxMonths | integer | No | |
| move_in_date | moveInDate | date | No | YYYY-MM-DD |
| deposit / security_deposit | depositAmount | integer | No | Whole dollars |
| utilities_included | utilitiesIncluded | string (comma-sep) | No | e.g. "water,trash,gas" |
| sqft / square_footage | squareFootage | integer | No | |
| year_built | yearBuilt | integer | No | |
| neighborhood_description | neighborhoodDescription | string | No | |
| url | url | string (uri) | No | External listing URL |
| internal_notes | internalNotes | string | No | Not shown to prospects |
| amenities | amenities | string (comma-sep) | No | |
| is_public | isPublic | boolean | No | true/false |
| owner_id | ownerId | uuid | No | Owner contact ID |

## API Bulk Endpoint

`POST /api/v1/properties/bulk`

Requires Pro or Scale plan. Max 500 properties per request. Async processing.

### Request

```json
{
  "properties": [
    { "street": "123 Main St", "rent": 1500, "beds": 2, "baths": 1, "city": "Austin" },
    { "address": "456 Oak Ave", "monthlyRent": 2000, "bedrooms": 3, "bathrooms": 2 }
  ]
}
```

Field names are flexible — use camelCase (`monthlyRent`) or common alternatives (`rent`, `beds`, `street_address`, `monthly_rent`). Same value coercion as dashboard import.

Supports `Idempotency-Key` header.

### Response (202 Accepted)

```json
{
  "data": {
    "jobId": "550e8400-e29b-41d4-a716-446655440000",
    "status": "pending",
    "total": 2
  }
}
```

### Poll Job Status

`GET /api/v1/properties/bulk/:jobId`

```json
{
  "data": {
    "jobId": "550e8400-...",
    "status": "completed",
    "total": 100,
    "created": 95,
    "failed": 5,
    "createdPropertyIds": ["id1", "id2", "..."],
    "unmappedFields": ["custom_field_x", "internal_id"],
    "errors": [
      { "row": 3, "field": "monthlyRent", "message": "Required", "code": "validation" },
      { "row": 98, "message": "Property limit exceeded (100 max for pro plan)", "code": "capacity" }
    ],
    "createdAt": "2026-02-19T10:00:00.000Z",
    "completedAt": "2026-02-19T10:00:05.000Z"
  }
}
```

Job statuses: `pending`, `processing`, `completed`, `failed`

You can also subscribe to `bulk_import.completed` webhook event.

### Error Codes

| Code | Description |
|------|-------------|
| validation | Field validation failed (missing required, invalid type) |
| capacity | Property limit exceeded for plan |
| duplicate | Duplicate address detected |

## Tips

- Export from AppFolio, Buildium, or any PMS as CSV
- Include at least Address, Rent, Bedrooms, Bathrooms
- More fields = better AI agent answers
- For step-by-step export instructions per platform, see the Property Import Guide in the dashboard
