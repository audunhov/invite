# Design Spec: Basic Auth and Cookie-based Sessions

## Overview
Implement a secure authentication system using email and password, hashed with bcrypt. Session management will be handled via a database-backed session table and secure HTTP-only cookies. The system will include a forgotten password flow using the existing SMTP infrastructure.

## Goals
- Secure administrative access to the system.
- Implement persistent sessions that survive server restarts.
- Add rate limiting to sensitive endpoints.
- Prevent accidental lockout by protecting the last administrative user.

## Architecture

### 1. Data Model Changes
- **Table `persons`**:
  - `password_hash`: `TEXT` (Nullable, contains bcrypt hash).
  - `password_reset_token`: `TEXT` (Nullable, unique).
  - `password_reset_expires_at`: `TIMESTAMP WITH TIME ZONE` (Nullable).
- **Table `sessions`**:
  - `id`: `UUID` (Primary Key).
  - `person_id`: `UUID` (Foreign Key to `persons.id`).
  - `expires_at`: `TIMESTAMP WITH TIME ZONE` (Not Null).
  - `created_at`: `TIMESTAMP WITH TIME ZONE` (Default `NOW()`).

### 2. Backend Implementation (Go)

#### Auth Package (`internal/auth`)
- `HashPassword(password string) (string, error)`: Uses `bcrypt.GenerateFromPassword`.
- `CheckPassword(password, hash string) bool`: Uses `bcrypt.CompareHashAndPassword`.
- `GenerateSecureToken() string`: Uses `crypto/rand` to generate a secure random string.

#### Rate Limiter (`internal/limiter`)
- An in-memory rate limiter (e.g., using `golang.org/x/time/rate`) keyed by IP address.
- Applied to `/api/auth/login`, `/api/auth/forgot-password`, and `/api/respond/*`.

#### Middleware
- `AuthMiddleware`: 
  - Extracts `session_id` from cookie.
  - Checks if session exists and is not expired in the DB.
  - Fetches the `Person` and injects it into the request context.
  - Returns `401 Unauthorized` if invalid.

#### API Endpoints
- `POST /api/auth/login`: Verifies email/password, creates session, sets `session_id` cookie.
- `POST /api/auth/logout`: Deletes session from DB, clears cookie.
- `POST /api/auth/forgot-password`: Generates reset token, saves to DB, sends email.
- `POST /api/auth/reset-password`: Verifies token, updates password, clears token.
- `GET /api/auth/me`: Returns the current logged-in user's details.

### 3. Frontend Implementation (Vue 3)
- **New Views**:
  - `LoginView.vue`: Basic login form.
  - `ForgotPasswordView.vue`: Request reset form.
  - `ResetPasswordView.vue`: New password form.
- **Router**:
  - Add `meta: { requiresAuth: true }` to internal routes.
  - Implement a `beforeEach` guard to check auth state (via a `GET /api/auth/me` check or a reactive store).
  - Redirect to `/login` if unauthorized.
- **HTTP Client**:
  - Add an interceptor to handle `401` responses and redirect to the login page.

### 4. Safety Constraints
- **Delete Protection**: Before deleting a person, the application will check:
  - `SELECT COUNT(*) FROM persons WHERE password_hash IS NOT NULL`.
  - If the count is 1 AND the person to be deleted has a `password_hash`, the deletion will be rejected with a `400 Bad Request`.

## Success Criteria
- [ ] Users can log in with a valid password and receive a secure cookie.
- [ ] Protected routes return `401` without a valid session.
- [ ] The "Forgotten Password" email is sent and the reset link works.
- [ ] The system remains accessible through the last admin user.
- [ ] Rate limiting triggers after a small number of rapid attempts.
