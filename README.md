# korauth — OpenKor Identity & Access Management

A standalone, microservice-based IAM (Identity & Access Management) service for the OpenKor cloud platform. Built with Go, PostgreSQL, and Redis.

## Overview

**korauth** is the authentication and authorization backbone of OpenKor. It provides:

- **JWT-based authentication** (RS256) with token refresh
- **Multi-tenant user management** with role-based access control (RBAC)
- **Public key distribution** via JWKS endpoint for downstream service integration
- **Brute-force protection** and secure password hashing (argon2id)
- **Soft-delete audit trail** for compliance

All external traffic flows through the `korapi` gateway; this service is never directly exposed.

## Quick Links

- **API Documentation**: Visit `/docs` endpoint (Redoc UI)
- **OpenAPI Specification**: [`api/openapi.yaml`](api/openapi.yaml)
- **JWT Technical Contract**: [`docs/TOKEN.md`](docs/TOKEN.md)
- **Related Services**: [OpenKor Project](https://github.com/OpenKorProject)

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.25+ (for local development)
- PostgreSQL 16+ (if running outside Docker)
- Redis 7+ (if running outside Docker)

### Using Docker Compose

```bash
# Clone and enter the repository
git clone <repo>
cd korauth

# Generate RSA keys
mkdir -p keys
openssl genrsa -out keys/jwt-private.pem 4096
openssl rsa -in keys/jwt-private.pem -pubout -out keys/jwt-public.pem

# Copy and configure environment
cp .env.example .env
# Edit .env if needed (for local dev, defaults usually work)

# Start services
docker compose up --build
```

**Output will show:**
```
==================================================
✓ KORAUTH INITIALIZED
==================================================
Admin Username: admin
Admin Password: changeme
Tenant ID:      <uuid>
==================================================
Login: POST http://localhost:8081/v1/auth/login
Docs:  http://localhost:8081/docs
==================================================
```

Visit `http://localhost:8081/docs` to explore the API with Redoc.

### Local Development

```bash
# Install dependencies
go mod download

# Create database & Redis (Docker or local)
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=changeme postgres:16-alpine
docker run -d -p 6379:6379 redis:7-alpine

# Run migrations and start server
go run ./cmd/korauth/main.go
```

## API Overview

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/v1/auth/login` | None | Login with username/password → access + refresh token |
| POST | `/v1/auth/refresh` | Refresh token | Get new access token |
| POST | `/v1/auth/logout` | Bearer | Revoke refresh token |
| GET | `/.well-known/jwks.json` | None | Public keys for token verification |
| GET | `/v1/auth/me` | Bearer | Current user info |
| GET/POST | `/v1/auth/users` | Admin | List / create users |
| GET/PATCH/DELETE | `/v1/auth/users/{id}` | Admin | User CRUD |
| PUT | `/v1/auth/users/{id}/roles` | Admin | Assign roles |
| GET/POST | `/v1/auth/tenants` | Admin | Tenant CRUD |

**Full API docs:** Visit `/docs` endpoint or see `api/openapi.yaml`

## Architecture

### Stack

- **Language:** Go 1.25
- **Web Framework:** Gin
- **Database:** PostgreSQL (schema: `auth`)
- **Cache/Sessions:** Redis
- **Auth:** RS256 JWT, argon2id password hashing
- **Monitoring:** Structured JSON logging (slog)

### Service Architecture

![System Architecture](docs/diagrams/korauth-architecture.png)

Shows 4-tier architecture:
- **Client Layer** → API Gateway → Service Layer → Infrastructure
- **korauth** (central service) with PostgreSQL, Redis, and JWKS endpoint
- **Other services** (korvm, korvol) validating tokens via JWKS
- Clear data flow: requests, queries, token verification

### Authentication Flow

![JWT Auth Flow](docs/diagrams/jwt-auth-flow.png)

Two parallel flows:
- **Left:** Login flow — POST /login → Verify credentials → Generate JWT → Store & return tokens
- **Right:** Validation flow — Service request with JWT → Fetch JWKS → Verify signature → Grant access
- Shows interaction with Redis (refresh tokens) and PostgreSQL (user data)

### Project Structure

```
korauth/
├── cmd/
│   └── korauth/
│       └── main.go              # Entry point
├── internal/
│   ├── config/                  # Configuration loading
│   ├── db/                       # Database & migrations
│   ├── handler/                  # HTTP handlers + Redoc UI
│   ├── middleware/               # Auth & RBAC middleware
│   ├── model/                    # Data models
│   ├── password/                 # Password hashing & validation
│   ├── redis/                    # Redis client
│   ├── seed/                     # Initial data seeding
│   ├── server/                   # HTTP server setup
│   ├── service/                  # Business logic
│   ├── store/                    # Database access layer
│   └── token/                    # JWT generation & parsing
├── api/
│   └── openapi.yaml             # OpenAPI 3.1 specification
├── docs/
│   └── TOKEN.md                 # JWT contract (for all services)
└── docker-compose.yml           # Local dev environment
```

## Configuration

### Environment Variables

| Variable | Default | Required | Notes |
|----------|---------|----------|-------|
| `DATABASE_URL` | — | Yes | PostgreSQL connection string |
| `REDIS_URL` | — | Yes | Redis connection string |
| `JWT_PRIVATE_KEY_PATH` | — | Yes | Path to RSA private key (4096 bit) |
| `JWT_PUBLIC_KEY_PATH` | — | Yes | Path to RSA public key |
| `ACCESS_TOKEN_TTL` | `15m` | No | Access token expiry |
| `REFRESH_TOKEN_TTL` | `168h` | No | Refresh token expiry (7 days) |
| `SEED_ADMIN_USERNAME` | `admin` | No | Initial admin username |
| `SEED_ADMIN_PASSWORD` | — | Yes | Initial admin password |
| `SEED_TENANT_NAME` | `default` | No | Initial tenant name |

### JWT Specification

See `docs/TOKEN.md` for the complete JWT contract, including:
- Claim structure and validation
- Key rotation procedures
- Brute-force protection
- Password policy (min. 8 chars, 1 uppercase, 1 digit, 1 special char)

## Development

### Running Tests

```bash
go test ./...
```

### Code Quality

```bash
# Format
go fmt ./...

# Lint
go vet ./...
golangci-lint run  # if installed
```

### Database Migrations

Migrations are embedded in the binary and auto-run on startup. To add a new migration:

1. Create `internal/db/migrations/NNN_description.up.sql`
2. Create matching `internal/db/migrations/NNN_description.down.sql`
3. Rebuild and run — migrations apply automatically

## Security Notes

- **Private keys** are never stored in version control — mount via Docker secrets or environment
- **Passwords** are hashed with argon2id (never logged or exposed)
- **Soft-delete** audit trail ensures compliance without data loss
- **Brute-force protection** rate-limits failed login attempts per tenant+username
- **RBAC** is enforced at the handler level (defense in depth)

## Deployment

### Docker

```bash
docker build -t korauth:latest .
docker run -p 8081:8081 \
  -e DATABASE_URL="..." \
  -e REDIS_URL="..." \
  -v /path/to/keys:/etc/openkor/keys:ro \
  korauth:latest
```

### Systemd (Linux)

For production Linux deployments without Docker, use systemd:

#### Installation

```bash
# Clone repository
git clone https://github.com/OpenKorProject/korauth
cd korauth

# Run installation script (requires root)
sudo bash systemd/install.sh
```

This script:
- Creates `korauth` system user & directories
- Builds and installs the binary to `/opt/korauth`
- Installs systemd service file
- Sets up proper file permissions

#### Configuration

**1. Generate RSA keys:**
```bash
sudo mkdir -p /etc/korauth/keys
sudo openssl genrsa -out /etc/korauth/keys/jwt-private.pem 4096
sudo openssl rsa -in /etc/korauth/keys/jwt-private.pem -pubout -out /etc/korauth/keys/jwt-public.pem
sudo chown korauth:korauth /etc/korauth/keys/*.pem
sudo chmod 600 /etc/korauth/keys/*.pem
```

**2. Configure environment:**
```bash
sudo nano /etc/korauth/korauth.env
```

Edit the following variables:
```bash
DATABASE_URL=postgres://user:password@localhost:5432/openkor?sslmode=require
REDIS_URL=redis://localhost:6379/0
SEED_ADMIN_PASSWORD=<strong-password>
```

**3. Ensure PostgreSQL and Redis are running:**
```bash
sudo systemctl status postgresql
sudo systemctl status redis-server
```

#### Operations

**Start/stop service:**
```bash
sudo systemctl start korauth
sudo systemctl stop korauth
sudo systemctl restart korauth
```

**Enable on boot:**
```bash
sudo systemctl enable korauth
```

**View status:**
```bash
sudo systemctl status korauth
```

**Follow logs:**
```bash
sudo journalctl -u korauth -f
```

**View logs from last 100 lines:**
```bash
sudo journalctl -u korauth -n 100
```

#### Directory Structure

| Path | Owner | Permissions | Purpose |
|------|-------|-------------|---------|
| `/opt/korauth/` | korauth:korauth | 755 | Binary location |
| `/etc/korauth/` | root:root | 755 | Configuration |
| `/etc/korauth/korauth.env` | root:root | 600 | Environment variables |
| `/etc/korauth/keys/` | korauth:korauth | 700 | RSA keys |
| `/var/lib/korauth/` | korauth:korauth | 755 | Runtime data |
| `/var/log/korauth/` | korauth:korauth | 755 | Logs (via journald) |

### Production Checklist

- [ ] Generate strong RSA keys (4096+ bit)
- [ ] Use a secret manager for sensitive env vars (AWS Secrets Manager, HashiCorp Vault, etc.)
- [ ] Enable HTTPS at the gateway level
- [ ] Configure database backups
- [ ] Set up log aggregation and monitoring
- [ ] Run database migrations before deploying
- [ ] Test token refresh flow end-to-end
- [ ] Verify JWKS endpoint is accessible to downstream services

## Troubleshooting

### Admin password forgotten

Use the `korauth-cli` admin utility to reset the password:

```bash
# Docker
docker compose exec korauth /app/korauth-cli reset-admin-password <tenant-id> <new-password>

# Systemd
/opt/korauth/korauth-cli reset-admin-password <tenant-id> <new-password>

# Example
/opt/korauth/korauth-cli reset-admin-password 6ec83570-ee9d-46b1-8a8c-f52a01ce987d NewPassword123!
```

**Note:** Password must meet policy requirements:
- Minimum 8 characters
- At least 1 uppercase letter
- At least 1 digit
- At least 1 special character

### Port already in use
```bash
# Check what's using port 8081
lsof -i :8081
```

### Database migration fails
```bash
# Check schema exists
psql $DATABASE_URL -c "SELECT * FROM auth.schema_migrations;"
```

### Token validation fails downstream
- Verify JWKS endpoint is accessible: `curl http://korauth:8081/.well-known/jwks.json`
- Check token not expired: `jq -R 'split(".")[1] | @base64d | fromjson' <(echo $TOKEN)`
- Verify issuer matches: should be `openkor-auth`

## Contributing

Contributions follow OpenKor conventions:

- **Commits:** Use conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`)
- **Testing:** Domain logic requires unit tests; mock external dependencies
- **Code style:** `go fmt`, `go vet`, `golangci-lint`
- **API changes:** Update `api/openapi.yaml` before implementation

## Related Projects

- **korapi** — API Gateway (routes `/v1/{service}/*` to appropriate microservice)
- **korvm** — VM management service
- **korvol** — Volume/storage service
- **korcli** — CLI client for OpenKor

## License

Apache 2.0 — See LICENSE file
# korauth
