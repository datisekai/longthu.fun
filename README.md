# Longthu.fun

Vietnamese badminton bill-splitter SaaS. Hosts finalize a session, players open a short link, pay via dynamic VietQR, the system reconciles automatically.

**Local folder name:** `dbadminton` (project codename). **Public domain:** `longthu.fun`.

## Dev quickstart

```bash
docker compose up -d mysql
cd backend && air            # backend on :8080
cd ../frontend && pnpm dev   # frontend on :5173
```

Backend healthcheck: `curl http://localhost:8080/healthz` → `{"status":"ok","version":"<git-sha>"}`.

## Stack

- **Frontend:** Rsbuild + React 19 + TypeScript + Tailwind v4 + shadcn/ui + TanStack Router + TanStack Query
- **Backend:** Go 1.22+ + Gin + sqlc + golang-migrate + air
- **Database:** MySQL 8.4 (utf8mb4 / utf8mb4_0900_ai_ci collation for Vietnamese diacritic-aware search)
- **Container:** Docker Compose for dev; single-VPS Caddy + Docker for prod
- **Payment:** payOS only in MVP (MBBank-gated for Auto-Detect)

## Planning & architecture

All planning artifacts live in [`_bmad-output/`](./_bmad-output/) — committed alongside code for traceability.

- [PRD](_bmad-output/planning-artifacts/prds/prd-dbadminton-2026-05-23/prd.md)
- [Architecture](_bmad-output/planning-artifacts/architecture.md)
- [Epics & Stories](_bmad-output/planning-artifacts/epics.md)
- [UX DESIGN.md](_bmad-output/planning-artifacts/ux-designs/ux-dbadminton-2026-05-23/DESIGN.md)
- [UX EXPERIENCE.md](_bmad-output/planning-artifacts/ux-designs/ux-dbadminton-2026-05-23/EXPERIENCE.md)
- [Payment provider research](_bmad-output/planning-artifacts/research/technical-payment-provider-comparison-for-longthufun-research-2026-05-23.md)
- [payOS integration follow-ups](_bmad-output/planning-artifacts/research/payos-integration-followups-2026-05-24.md)

## Dev backlog

Story-by-story tracking in [`_bmad-output/implementation-artifacts/`](./_bmad-output/implementation-artifacts/). Each story file is self-sufficient and includes acceptance criteria + tasks.

## License

TBD.
