# Longthu.fun - Badminton Court Bill Sharing Platform

Hệ thống chia bill cầu lông tự động. Host tạo buổi chơi → chia tiền → player trả qua QR.

## Quick Start

### 1. Prerequisites

- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- pnpm

### 2. Setup payOS Credentials

Đăng ký tài khoản payOS tại [https://my.payos.vn](https://my.payos.vn) để lấy:
- `PAYOS_CLIENT_ID`
- `PAYOS_API_KEY`
- `PAYOS_CHECKSUM_KEY`

### 3. Run với Docker

```bash
# Copy và edit .env (điền payOS credentials)
cp backend/.env.example backend/.env

# Chạy MySQL + Backend + Frontend
docker compose --profile full up -d

# Apply migrations
docker compose exec backend make migrate-up
```

Truy cập: http://localhost:5173

### 4. Run Backend (Local dev)

```bash
# Terminal 1: MySQL
docker compose up mysql -d

# Terminal 2: Backend
cd backend
cp .env.example .env
# Edit .env với credentials của bạn
go install github.com/air-verse/air@latest
make migrate-up
make run
# Server chạy tại http://localhost:8080

# Terminal 3: Frontend
cd frontend
pnpm install
pnpm dev
# App chạy tại http://localhost:5173
```

### 5. Run Frontend (Local dev)

```bash
cd frontend
pnpm install
pnpm dev
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | MySQL connection string | ✅ |
| `JWT_SECRET` | JWT signing secret (generate: `openssl rand -hex 32`) | ✅ |
| `SECRETS_MASTER_KEY` | AES encryption key for payOS credentials | ✅ |
| `PAYOS_CLIENT_ID` | payOS Client ID | ✅ cho Auto-Detect |
| `PAYOS_API_KEY` | payOS API Key | ✅ cho Auto-Detect |
| `PAYOS_CHECKSUM_KEY` | payOS Checksum Key | ✅ cho Auto-Detect |
| `APP_BASE_URL` | Frontend URL (default: http://localhost:5173) | Optional |
| `PORT` | Backend port (default: 8080) | Optional |

## Architecture

```
frontend/          # React 19 + Rsbuild + shadcn/ui
backend/
  cmd/api/         # HTTP entry point
  internal/
    auth/         # JWT authentication
    dashboard/     # Host dashboard
    groups/        # Group management
    players/       # Player management
    sessions/      # Session & charge management
    paymentintents/ # Payment intent creation
    payments/      # Payment matching & webhook handling
    webhooks/      # payOS webhook endpoints
    autodetect/    # Auto-detect setup & management
    public/        # Public endpoints (/g/, /p/, /pay/)
```

## Key Endpoints

### Public (no auth)
- `GET /api/v1/group-bill/:shareCode` - Group bill page
- `GET /api/v1/player-ledger/:playerCode` - Player ledger
- `POST /api/v1/payment-intents` - Create payment intent
- `GET /api/v1/payment-intents/:code` - Get payment intent status
- `POST /api/v1/webhooks/payos` - payOS webhook

### Authenticated
- `POST /api/v1/auth/register` - Register host
- `POST /api/v1/auth/login` - Login
- `GET /api/v1/groups` - List groups
- `POST /api/v1/sessions` - Create session
- `PATCH /api/v1/sessions/:id/finalize` - Finalize session
- `GET /api/v1/dashboard` - Host dashboard

## Testing Flow

1. **Register/Login** → Tạo host account
2. **Add Bank Account** → Thêm tài khoản ngân hàng (MBBank recommended)
3. **Create Group** → Tạo group cho buổi chơi
4. **Add Players** → Thêm players vào group
5. **Create Session** → Tạo buổi với cost items
6. **Finalize** → Chốt bill, generate share code
7. **Share Link** → Copy link gửi cho players
8. **Player Pays** → Player vào `/p/:code` → tạo QR → chuyển tiền
9. **Host Confirms** → Host đánh dấu đã nhận tiền

## Development

```bash
# Backend
make build          # Build
make test           # Run tests
make sqlc-generate  # Regenerate DB queries

# Frontend
pnpm build          # Production build
pnpm test           # Run tests
```

## License

MIT
