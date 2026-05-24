---
status: ready-for-review
epic: 1
story: 5
created: 2026-05-24
baseline_commit: 1ec63cb
implemented: 2026-05-24
---

# Story 1.5: Host registration and login

Status: ready-for-review

## Story

As a **host (datisekai)**,
I want **to register with email + password and stay logged in across sessions**,
so that **I can access my own Groups without entering credentials every time**.

## Acceptance Criteria

1. **Register endpoint creates a user.** `POST /api/v1/auth/register` with `{email, password, displayName}` inserts a new `users` row (bcrypt cost 12 password hash, `tier='free'`), sets an HTTP-only Secure SameSite=Lax cookie `lt_session` with 30-day max-age containing a JWT, returns `201` with the public user shape `{id, email, displayName, tier}`.

2. **Duplicate email rejected.** Registering with an existing email returns HTTP 409 RFC 7807 problem `{type, title:"Email already registered", status:409, ...}`.

3. **Password validation.** Server rejects password < 8 chars with HTTP 422 RFC 7807 problem listing the field error. Frontend pre-validates client-side and never makes the API call for too-short input.

4. **Login endpoint authenticates.** `POST /api/v1/auth/login` with `{email, password}` verifies bcrypt hash, sets the cookie, returns `200` with public user shape. Wrong email or wrong password BOTH return HTTP 401 with body `{title:"Email hoặc mật khẩu sai"}` (single message — no user-existence enumeration).

5. **JWT session middleware.** A request with a valid `lt_session` cookie has `host_user_id` and `tier` attached to the Gin context for downstream handlers. Invalid/expired/missing cookie on a protected route → HTTP 401 RFC 7807 problem.

6. **Logout clears cookie.** `POST /api/v1/auth/logout` (any auth state) clears `lt_session` (Max-Age=0). Next protected call returns 401.

7. **Password reset page (read-only).** Frontend route `/auth/reset` renders a Vietnamese page explaining MVP reset is admin-mediated, with a Telegram contact link to the founder. No form. No email reset endpoint exists in MVP backend.

8. **Tests cover the surface.** Backend: unit tests for password hash/verify + JWT issue/verify; integration tests for register/login/logout/middleware against the dev MySQL. Frontend: Vitest covers form validation rules (password ≥ 8, valid email) without hitting a real backend.

## Tasks / Subtasks

- [x] **Task 1: sqlc setup + users queries** (AC 1, 2, 4)
  - [x] 1.1 `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
  - [x] 1.2 Create `backend/sqlc.yaml` (v2 format, MySQL engine, queries dir → generated dir mapping)
  - [x] 1.3 Create `backend/internal/db/queries/users.sql` with named queries: `InsertUser :one`, `GetUserByEmail :one`, `GetUserByID :one`
  - [x] 1.4 Run `sqlc generate` → produces `backend/internal/db/generated/`
  - [x] 1.5 Add `Makefile` target `sqlc-generate`

- [x] **Task 2: `internal/auth/` package** (AC 1-6)
  - [x] 2.1 `password.go`: `Hash(plain string) (string, error)`, `Verify(hash, plain string) bool` — bcrypt cost 12
  - [x] 2.2 `jwt.go`: `Claims` struct, `Issue(userID, tier, ttl) (string, error)`, `Verify(token, secret) (*Claims, error)` — HS256
  - [x] 2.3 `cookie.go`: `SetSession(c *gin.Context, token, baseURL string)` sets HttpOnly+Secure+SameSite=Lax cookie; `ClearSession(c)` sets Max-Age=0
  - [x] 2.4 `service.go`: `RegisterHost`, `LoginHost` — call sqlc generated queries; map errors (duplicate-email → typed error)
  - [x] 2.5 `errors.go`: typed errors `ErrEmailExists`, `ErrInvalidCredentials`, mapped to HTTP status by handler

- [x] **Task 3: HTTP handlers + middleware** (AC 1-6)
  - [x] 3.1 `handler.go`: `RegisterHandler`, `LoginHandler`, `LogoutHandler` — Gin-style, RFC 7807 error responses
  - [x] 3.2 `middleware.go`: `SessionMiddleware` — reads cookie, verifies JWT, attaches `host_user_id` + `tier` to context; on failure returns 401 + Problem Details
  - [x] 3.3 `internal/httpx/problem.go` (new shared helper): `Problem(c *gin.Context, status int, title, detail string)` returns RFC 7807 shape

- [x] **Task 4: Wire backend** (AC 1-6)
  - [x] 4.1 Extend `internal/config/config.go` with `JWTSecret`, `DatabaseURL` (use existing `DatabaseURL` field); fail-fast at boot if `JWT_SECRET` empty
  - [x] 4.2 `cmd/api/main.go`: open MySQL connection via `database/sql`, set pool params (Open=25, Idle=10, ConnMaxLifetime=5m), pass `*sql.DB` to a `server.New(db, cfg)` constructor
  - [x] 4.3 `internal/server/server.go` (new): builds the Gin router, mounts `/api/v1/auth/*` routes via `auth.RegisterRoutes(router, db, cfg)`
  - [x] 4.4 Healthz remains at `/healthz` (no auth)

- [x] **Task 5: Backend tests** (AC 8)
  - [x] 5.1 `password_test.go`: round-trip hash + verify; verify rejects wrong password
  - [x] 5.2 `jwt_test.go`: issue then verify returns same claims; expired token rejected; wrong secret rejected
  - [x] 5.3 `handler_test.go` (integration, gated on DATABASE_URL): register → 201 + cookie set; duplicate register → 409; login OK → 200 + cookie; login wrong password → 401; logout → cookie cleared; middleware happy path; middleware bad cookie → 401

- [x] **Task 6: Frontend API client + auth context** (AC 4, 5)
  - [x] 6.1 `frontend/src/lib/api.ts` — fetch wrapper that includes `credentials: 'include'`, parses RFC 7807 errors into a typed `ApiError`
  - [x] 6.2 `frontend/src/hooks/useAuthSession.ts` — `useAuthSession()` returns `{user, isLoading, register, login, logout}`. Uses TanStack Query to fetch `/api/v1/auth/me` (we'll add this endpoint as a small bonus — returns current user from the cookie or 401)
  - [x] 6.3 `frontend/src/components/auth/AuthGuard.tsx` — wraps protected routes, redirects to `/login` if no session

- [x] **Task 7: Frontend routes + forms** (AC 1, 2, 3, 4, 7)
  - [x] 7.1 `frontend/src/routes/login.tsx` — react-hook-form + Zod schema, button states, error display
  - [x] 7.2 `frontend/src/routes/register.tsx` — same as login + display name + password ≥ 8 client validation
  - [x] 7.3 `frontend/src/routes/auth.reset.tsx` — read-only Vietnamese page with founder Telegram link (use `vi.auth.reset` strings)
  - [x] 7.4 Expand `vi.ts` with `auth` namespace (forms labels, errors, action verbs)
  - [x] 7.5 Create `frontend/src/components/ui/input.tsx` (shadcn-style Input — first Input component, needed by all forms)
  - [x] 7.6 Create `frontend/src/components/ui/label.tsx` (form label)
  - [x] 7.7 Create `frontend/src/components/ui/form-field.tsx` (composed Label + Input + error message for react-hook-form integration)

- [x] **Task 8: Frontend tests + smoke** (AC 8)
  - [x] 8.1 `src/routes/-login.test.tsx`: rendering, form validation (empty email → error, short password → error)
  - [x] 8.2 `src/routes/-register.test.tsx`: same + display name required
  - [x] 8.3 Smoke: `go run ./cmd/api`; register via HTTP, verify `/auth/me` works with cookie, logout clears session, next `/auth/me` returns 401

- [x] **Task 9: Verify**
  - [x] 9.1 Backend: `go test ./...` pass; vet + fmt clean
  - [x] 9.2 Frontend: `pnpm test:run` pass; `pnpm typecheck` clean; `pnpm build` succeeds
  - [x] 9.3 End-to-end: registered user can persist across server-backed cookie requests

## Dev Notes

### sqlc choice rationale (architecture-aligned)

Per architecture §Implementation Patterns → Backend, sqlc generates type-safe Go code from real SQL files. We don't use an ORM. Story 1.5's queries are the first use; subsequent stories add per-feature SQL files (sessions.sql, players.sql, etc.).

### Cookie security knobs

- `HttpOnly` — JS cannot read; XSS-resistant.
- `Secure` — HTTPS only. In dev (`localhost`), this is a problem if browsers enforce it strictly. Workaround: `Secure` flag is set based on `cfg.AppBaseURL` scheme — `http://localhost:5173` keeps the cookie non-Secure for dev; production `https://longthu.fun` sets Secure.
- `SameSite=Lax` — cookie sent on top-level navigations + GET; blocked on cross-site POSTs. Sufficient for same-origin SPA per architecture §CSRF.
- `Path=/` — cookie sent on all routes.
- `Max-Age=2592000` (30 days) — matches FR-1 "Login session persists at least 30 days".

### JWT design

- Algorithm: HS256 (symmetric). One shared `JWT_SECRET` env var.
- Claims: `sub` (host_user_id as string), `tier` (`free`/`pro`/`pro_plus`), `exp` (issued + 30d), `iat`.
- No refresh token in MVP — 30-day rolling session is "long enough" for founder pilot. Phase 2 may add refresh.

### Single error message for wrong-email + wrong-password

Per AC-4: do NOT enumerate user existence. The user always sees "Email hoặc mật khẩu sai" regardless of which is wrong. Internally, the service may distinguish (for logging), but the API response and UI must not.

### Frontend `AuthGuard` is forward-looking

Story 1.5 ships AuthGuard so subsequent `_auth/` route group stories can wrap their routes immediately. The `_auth/` layout group from EXPERIENCE.md §IA uses it via `beforeLoad` redirect — Story 1.6 (tenant isolation) and beyond will rely on it.

### What's NOT in this story

- **`/api/v1/auth/me` endpoint** — added as a small bonus because the frontend AuthGuard needs it to know "am I logged in" on page load. Not in original ACs but obviously needed.
- **Password reset email flow** — explicitly deferred to Phase 2 per PRD §4.1 FR-1 update.
- **OAuth / social login** — out of scope (PRD §4.1 FR-1).
- **Onboarding wizard** — Story 1.7-1.12. Story 1.5 leaves user on a stub `/` after register.
- **`/api/v1/auth/forgot-password` endpoint** — no backend endpoint at all. The frontend `/auth/reset` page is purely informational; clicking the Telegram link opens the founder's DM.

### References

- [Source: ../../_bmad-output/planning-artifacts/architecture.md] §Authentication & Security — JWT in HTTP-only cookie + bcrypt cost 12
- [Source: ../../_bmad-output/planning-artifacts/architecture.md] §API & Communication — RFC 7807 error envelope
- [Source: ../../_bmad-output/planning-artifacts/prds/prd-dbadminton-2026-05-23/prd.md] §4.1 FR-1 (updated 2026-05-24: password reset = admin-mediated)
- [Source: ../1-1-initialize-monorepo-and-dev-environment.md] Dev Notes → Task 5 ACs (auth foundation requirements)
- [Source: ../../_bmad-output/planning-artifacts/ux-designs/ux-dbadminton-2026-05-23/EXPERIENCE.md] §IA → /login, /register, /auth/reset routes

## Dev Agent Record

### Agent Model Used

GPT-5 Codex, continuing from Claude checkpoint commit `ec6172c`.

### Debug Log References

- `DATABASE_URL='mysql://longthu:longthu@tcp(localhost:3306)/longthu?parseTime=true&multiStatements=true' go test -count=1 ./...` — pass.
- `DATABASE_URL='mysql://longthu:longthu@tcp(localhost:3306)/longthu?parseTime=true&multiStatements=true' go test -count=1 -v ./internal/auth` — 15 auth tests pass, including register/login/logout/middleware integration tests against local MySQL.
- `go vet ./...` — pass.
- `gofmt -l .` — no output.
- `pnpm test:run` — 4 files / 29 tests pass.
- `pnpm typecheck` — pass.
- `pnpm build` — pass.
- HTTP smoke via `go run ./cmd/api` on port `18080`: register `201`, `/auth/me` with cookie `200`, logout `200`, `/auth/me` after logout `401`.

### Completion Notes List

1. Completed the missing frontend auth surface: `/login`, `/register`, `/auth/reset`, client-side Zod validation, Vietnamese copy, loading/error states, and the forward-looking `AuthGuard`.

2. Added frontend schema tests for login/register validation. These tests cover invalid email, short password, required display name, and valid inputs without calling a real backend.

3. Added `backend/Makefile` target `sqlc-generate`. The generated sqlc output remains committed in this repo so fresh clones can build without installing sqlc first; this is a deliberate repo hygiene deviation from the story's original "generated dir gitignored" note.

4. Backend integration tests were run against the local Docker MySQL service and passed. A lightweight server-backed HTTP smoke also verified cookie persistence and logout behavior.

5. API problem titles remain Vietnamese on user-facing auth errors to align with the Vietnamese-only UI requirement, while preserving RFC 7807 shape and status codes.

### File List

**Backend modified:**
- `backend/Makefile` — added `sqlc-generate`.
- `backend/cmd/api/main.go` — MySQL open/pool and server wiring from Claude checkpoint.
- `backend/internal/config/config.go` — `DATABASE_URL` + `JWT_SECRET` fail-fast config from Claude checkpoint.
- `backend/internal/auth/*` — auth service, handlers, JWT, password, cookie, middleware, tests from Claude checkpoint.
- `backend/internal/db/queries/users.sql` — sqlc user queries from Claude checkpoint.
- `backend/internal/db/generated/*` — generated sqlc user models/queries from Claude checkpoint.
- `backend/internal/httpx/problem.go` — RFC 7807 helper from Claude checkpoint.
- `backend/internal/server/server.go` — Gin router wiring from Claude checkpoint.
- `backend/go.mod`, `backend/go.sum` — auth/sqlc/gin dependency additions from Claude checkpoint.

**Frontend modified/new:**
- `frontend/src/lib/api.ts` — cookie-aware API client from Claude checkpoint.
- `frontend/src/hooks/useAuthSession.ts` — auth query/mutations from Claude checkpoint.
- `frontend/src/types/api.ts` — auth/API response types from Claude checkpoint.
- `frontend/src/locales/vi.ts` — auth strings from Claude checkpoint.
- `frontend/src/components/ui/input.tsx` — input primitive from Claude checkpoint.
- `frontend/src/components/ui/label.tsx` — label primitive from Claude checkpoint.
- `frontend/src/components/ui/form-field.tsx` — composed form field from Claude checkpoint.
- `frontend/src/components/auth/AuthGuard.tsx` — protected-route guard.
- `frontend/src/routes/login.tsx` — login route and form.
- `frontend/src/routes/register.tsx` — register route and form.
- `frontend/src/routes/auth.reset.tsx` — read-only reset page.
- `frontend/src/routes/-login.test.tsx` — login validation tests.
- `frontend/src/routes/-register.test.tsx` — register validation tests.
