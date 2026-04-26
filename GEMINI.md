# Project Guide for Gemini CLI

This document contains essential information for navigating and developing this project.

## Core Mandates
- **Always Run Seed After Reset:** If the database is empty or reset, use the seed utility immediately.
- **Strict Type Safety:** The frontend uses `openapi-fetch` and generated types. Never use raw `fetch` for API calls.
- **Pre-commit Hooks:** Lefthook is installed. Ensure `lefthook run pre-commit` passes before concluding work.

## Development Utilities

### Database Seeding
Populate the database with an admin user, sample persons, groups, tags, and invitations.
```bash
docker compose exec app go run main.go frontend.go seed
```
*Note: Uses `TRUNCATE CASCADE` - will wipe existing data.*

### Email Capture (Mailhog)
All outgoing emails are captured locally.
- **Web UI:** http://localhost:8025
- **SMTP Port:** 1025 (no auth required in local dev)

### End-to-End Testing
Verified the "Golden Path" (Login -> Create Invite -> Dashboard).
```bash
cd frontend && npm run test:e2e
```

## Architecture Notes

### Frontend API Client
Centralized client located at `frontend/src/utils/api.ts`.
- **Library:** `openapi-fetch`
- **Error Handling:** Global middleware automatically triggers toast notifications for non-OK responses.
- **Usage:** `import { client } from '@/utils/api'`

### Database Persistence
- **Volume:** `invite_db_data` persists PostgreSQL data across container restarts.
- **Location:** Managed by Docker Compose.

## Key Credentials (Local Dev)
- **Admin Email:** `admin@example.com`
- **Admin Password:** `password123`
