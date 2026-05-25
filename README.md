# Longthu.fun — Badminton Court Bill Sharing Platform

Hệ thống chia bill cầu lông tự động. Host tạo buổi chơi → chia tiền → player trả qua QR.

## Tech Stack

- **Frontend**: React 19, Rsbuild, shadcn/ui, TanStack Router, TanStack Query
- **Backend**: Go 1.25, Gin, sqlc, MySQL 8.4
- **Payments**: payOS ( VietQR )
- **Infrastructure**: Docker Compose

## Quick Start

### 1. Prerequisites

- Docker & Docker Compose
- Go 1.25+
- Node.js 22+
- pnpm 10+

### 2. Clone & Setup

```bash
git clone https://github.com/datisekai/longthu.fun.git
cd longthu.fun

# Copy env files
cp backend/.env.example backend/.env
```

### 3. Configure Environment

Edit `backend/.env`:

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | MySQL connection string | ✅ |
| `JWT_SECRET` | JWT signing secret (`openssl rand -hex 32`) | ✅ |
| `SECRETS_MASTER_KEY` | AES key for encrypting payOS credentials (32 chars) | ✅ |
| `APP_BASE_URL` | Frontend URL (e.g. https://longthu.fun) | ✅ |
| `PORT` | Backend port (default: 8080) | Optional |
| `ADMIN_EMAIL` | Admin account email | Optional |
| `ADMIN_PASSWORD` | Admin account password | Optional |
| `PAYOS_CLIENT_ID` | payOS Client ID | For auto-detect |
| `PAYOS_API_KEY` | payOS API Key | For auto-detect |
| `PAYOS_CHECKSUM_KEY` | payOS Checksum Key | For auto-detect |

### 4. Run with Docker

```bash
# Start all services (MySQL + Backend + Frontend)
docker compose --profile full up -d

# Apply database migrations
docker compose exec backend make migrate-up
```

Access: http://localhost:5173

### 5. Local Development

```bash
# Terminal 1: MySQL
docker compose up mysql -d

# Terminal 2: Backend
cd backend
go install github.com/air-verse/air@latest
make migrate-up
make run          # hot-reload via Air

# Terminal 3: Frontend
cd frontend
pnpm install
pnpm dev
```

## Database Migrations

```bash
cd backend
make migrate-create name=your_migration
# Edit migrations/xxxx_your_migration.up.sql
make migrate-up
make migrate-down
```

## Architecture

```
frontend/                    # React 19 + Rsbuild
  src/
    routes/                 # TanStack Router file-based routes
    components/             # Shared UI components
    hooks/                  # Custom React hooks
    lib/                    # API client, utils

backend/
  cmd/api/                  # HTTP entry point
  internal/
    auth/                   # JWT authentication
    admin/                  # Admin dashboard (tier management)
    autodetect/             # Auto-detect payOS setup
    bankaccounts/           # Host bank account management
    dashboard/               # Host dashboard API
    db/                     # sqlc generated code + SQL queries
    groups/                 # Group management
    payments/               # Payment matching logic
    players/                # Player management
    paymentintents/         # Payment intent creation
    public/                 # Public endpoints (group bill, player ledger)
    sessions/               # Session & charge management
    webhooks/               # payOS webhook endpoints
```

## Testing Flow

1. **Register/Login** → Tạo host account
2. **Add Bank Account** → Thêm tài khoản ngân hàng (MBBank recommended)
3. **Create Group** → Tạo group cho buổi chơi
4. **Add Players** → Thêm players vào group
5. **Create Session** → Tạo buổi với cost items
6. **Finalize** → Chốt bill, generate share code
7. **Share Link** → Copy link gửi cho players
8. **Player Pays** → Player vào `/p/:code` → tạo VietQR → chuyển tiền
9. **Host Confirms** → Host đánh dấu đã nhận tiền

## Development Scripts

```bash
# Backend
make build          # Build binary
make test           # Run tests
make sqlc-generate   # Regenerate DB queries
make migrate-up     # Run pending migrations
make migrate-down   # Rollback last migration

# Frontend
pnpm dev            # Dev server with hot-reload
pnpm build          # Production build
pnpm preview        # Preview production build
pnpm typecheck       # TypeScript check
pnpm lint           # ESLint
pnpm test           # Run tests
```

## CI/CD

GitHub Actions runs on every push/PR to `main`:

- **Backend**: go vet, build, tests, gitleaks scan
- **Frontend**: typecheck, lint, test, build, Lighthouse audit
- **Docker**: multi-stage build sanity check

## License

MIT
