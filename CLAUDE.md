# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

### Environment Setup
```bash
# Copy .env.example to .env and configure database credentials
cp .env.example .env

# Load environment variables
set -a; source .env; set +a
```

### Build & Run
```bash
# Build all packages
go build ./...

# Run the server (starts on port 8080 by default)
go run ./cmd/server

# Build for Linux deployment
GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/guangfu250923 ./cmd/server
```

### Database
- Migrations run automatically on server startup via `db.Migrate()`
- Uses PostgreSQL with pgx/v5 connection pool
- Creates tables only if they don't exist (idempotent)
- Migration code: `internal/db/migrate.go`

### OpenAPI Spec
```bash
# Lint the OpenAPI specification
spectral lint --ruleset .spectral.yaml openapi.yaml

# Swagger UI available at http://localhost:8080/swagger/index.html
```

## Architecture

### Project Structure
```
cmd/
├── server/              # Main API server (Gin + PostgreSQL)
├── updater/             # Self-updater service for automated deployments
└── import_accommodations/ # Data import utility

internal/
├── config/              # Environment configuration loader
├── db/                  # Database connection and migrations
├── handlers/            # HTTP handlers (one file per domain resource)
├── middleware/          # Request logging, CORS, cache headers, security, IP filtering
├── models/              # Domain models (structs matching database tables)
└── sheetcache/          # Google Sheets cache polling service
```

### HTTP Stack
Server uses Gin framework with middleware chain:
1. **CORS**: Configured for multiple frontend origins
2. **RequestLogger**: Logs all requests to `request_logs` table
3. **CacheHeaders**: Adds cache control headers for GET responses
4. **SecurityHeaders**: CSP and security headers
5. **IPFilter**: Blocks IPs based on `ip_denylist` table and country headers

Route registration: `cmd/server/main.go:88-148`

### Database Layer
- Connection pool management: `internal/db/conn.go`
- All timestamps stored as Unix epoch (int64) in JSON, `timestamptz` in PostgreSQL
- Coordinates stored as separate `lat`/`lng` columns, serialized as nested objects in JSON
- Array fields (facilities, services, etc.) use PostgreSQL `text[]`

## Key Design Patterns

### JSON-LD Pagination
All collection endpoints return standardized format:
```json
{
  "@context": "https://www.w3.org/ns/hydra/context.jsonld",
  "@type": "Collection",
  "totalItems": 123,
  "member": [...],
  "limit": 50,
  "offset": 0,
  "next": "/endpoint?limit=50&offset=50",
  "previous": null
}
```

### Supply/SupplyItem Relationship
- **Supply**: Warehouse/distribution point (name, address, phone, notes)
- **SupplyItem**: Individual supply items (tag, name, counts, unit) linked to Supply via `supply_id`
- Creating Supply can optionally embed one initial SupplyItem
- **Important naming**: Field is `recieved_count` (intentional typo for frontend compatibility), but database column is `received_count`
- Batch distribution: `POST /supplies/{id}` accumulates `recieved_count` with validation against `total_count`
- Handlers: `internal/handlers/supply_handlers.go`

### HumanResource Aggregate
`human_resources` table is a wide aggregate containing:
- Organization info (org, address, phone, status)
- Role details (role_name, role_type, skills, certifications)
- Headcount tracking (headcount_need, headcount_got, headcount_unit)
- Shift scheduling (shift_start_ts, shift_end_ts)
- Assignment tracking (assignment_timestamp, assignment_count)
- Aggregate statistics (total_roles, completed_roles, pending_roles, urgent_requests, etc.)

Uses composite primary key as text ID.

### Request Logging
All API requests logged to `request_logs` table with:
- Request details (method, path, query, headers, body)
- Response (status_code, duration_ms, error)
- Original/result data snapshots (JSONB)
- Resource ID tracking

View recent logs: `GET /_admin/request_logs`

### Spam Detection
`spam_result` table tracks LLM-based spam/malicious content detection:
- **id**: Unique identifier for the spam check result (string)
- **target_id**: ID of the resource being validated (e.g., human_resources or supplies UUID)
- **target_type**: Table name of the target resource (e.g., "human_resources", "supplies")
- **target_data**: Original data snapshot (JSONB) of the target resource
- **is_spam**: Boolean flag indicating if LLM detected spam/malicious content
- **judgment**: Text explanation of why LLM flagged it as spam (e.g., "警告語氣,可能為惡意")
- **validated_at**: Unix timestamp (stored as bigint) when LLM validation occurred

Used for automated content moderation on user-submitted data.

## Domain Resources

### Core Resources (CRUD Pattern)
Most resources follow standard pattern:
- `POST /{resource}` - Create
- `GET /{resource}` - List (paginated)
- `GET /{resource}/{id}` - Get single
- `PATCH /{resource}/{id}` - Partial update

Resources:
- `/shelters` - Shelter/evacuation centers
- `/medical_stations` - Medical support facilities
- `/mental_health_resources` - Mental health services
- `/accommodations` - Temporary housing
- `/shower_stations` - Shower/bathing facilities
- `/water_refill_stations` - Water refill points
- `/restrooms` - Restroom facilities
- `/volunteer_organizations` - Volunteer recruitment orgs
- `/human_resources` - Human resource needs/assignments
- `/supplies` + `/supply_items` - Supply warehouses and inventory
- `/reports` - Incident reports
- `/spam_results` - LLM spam detection results (query filters: target_type, target_id, is_spam)

## Deployment

Systemd service files and deployment guide: `deploy/README.md`
- `guangfu250923.service` - Main API service

Environment variables documented in `deploy/README.md` and `.env.example`.

**Note**: the `guangfu-updater.service` service is deprecated, and it CAN be IGNORED.
**Note**: the `Sheet Cache` related implementaions is also being deprecated, and it should be removed in the future.
