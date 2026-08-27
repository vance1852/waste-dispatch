# Waste Dispatch — 生活垃圾处理调度系统

A production-grade Go backend for managing waste collection dispatch operations.

## Features

- **User management** with role-based access control (admin / operator / driver / resident)
- **Vehicle fleet management** with state machine and optimistic locking
- **Collection point monitoring** with capacity tracking and auto-overflow detection
- **Collection task scheduling** with full lifecycle state machine (scheduled → in_progress → completed / failed)
- **Incident management** — report, assign, and resolve incidents
- **Resident credit system** with idempotent transactions
- **Audit logging** for every significant change
- **Background workers** for stale-task recovery and overflow detection
- **Graceful shutdown** with configurable timeout

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.22 |
| HTTP | Gin |
| Database | SQLite (WAL mode) |
| Migrations | golang-migrate |
| Logging | zerolog |

## Quick Start

```bash
# Copy environment file
cp .env.example .env

# Edit .env and set AUTH_TOKEN_SECRET to a long random string

# Run the server
make run
```

The server starts on `http://localhost:8080`.

## API Endpoints

### Authentication
| Method | Path | Description |
|---|---|---|
| POST | /api/v1/auth/register | Register a new user |
| POST | /api/v1/auth/login | Log in and get a token |
| POST | /api/v1/auth/logout | Revoke current session |
| GET | /api/v1/auth/me | Get current user profile |

### Vehicles
| Method | Path | Description |
|---|---|---|
| GET | /api/v1/vehicles | List vehicles |
| POST | /api/v1/vehicles | Create vehicle |
| GET | /api/v1/vehicles/:id | Get vehicle |
| PUT | /api/v1/vehicles/:id/assign | Assign driver |
| PUT | /api/v1/vehicles/:id/release | Release vehicle |
| DELETE | /api/v1/vehicles/:id | Delete vehicle |

### Collection Points
| Method | Path | Description |
|---|---|---|
| GET | /api/v1/points | List points |
| GET | /api/v1/points/over-capacity | List over-capacity points |
| POST | /api/v1/points | Create point |
| GET | /api/v1/points/:id | Get point |
| PUT | /api/v1/points/:id/load | Update load reading |
| DELETE | /api/v1/points/:id | Delete point |

### Collection Tasks
| Method | Path | Description |
|---|---|---|
| GET | /api/v1/tasks | List tasks |
| POST | /api/v1/tasks | Schedule task |
| GET | /api/v1/tasks/:id | Get task |
| PUT | /api/v1/tasks/:id/start | Start task |
| PUT | /api/v1/tasks/:id/complete | Complete task |
| PUT | /api/v1/tasks/:id/fail | Fail task |
| PUT | /api/v1/tasks/:id/cancel | Cancel task |

### Incidents
| Method | Path | Description |
|---|---|---|
| GET | /api/v1/incidents | List incidents |
| POST | /api/v1/incidents | Report incident |
| GET | /api/v1/incidents/:id | Get incident |
| PUT | /api/v1/incidents/:id/assign | Assign incident |
| PUT | /api/v1/incidents/:id/resolve | Resolve incident |

### Credits
| Method | Path | Description |
|---|---|---|
| GET | /api/v1/credits/:resident_id/balance | Get balance |
| GET | /api/v1/credits/:resident_id/transactions | List transactions |
| POST | /api/v1/credits/:resident_id/earn | Earn credits |
| POST | /api/v1/credits/:resident_id/redeem | Redeem credits |

### Health
| Method | Path | Description |
|---|---|---|
| GET | /health | Health check |

## Authentication

All protected endpoints require:
```
Authorization: Bearer <token>
```

## Docker

```bash
make docker-build
make docker-run
```

## Configuration

All configuration is via environment variables. See `.env.example` for the full list.

## Project Structure

```
waste-dispatch/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── config/                 # Configuration loading
│   ├── domain/                 # Core entities and business rules
│   ├── repository/             # Persistence interfaces + SQLite implementations
│   ├── service/                # Business logic services
│   ├── httpapi/                # HTTP server, handlers, response helpers
│   ├── middleware/             # Auth, logging, request-ID middleware
│   ├── storage/sqlite/         # Database connection + migrations
│   ├── worker/                 # Background workers
│   └── audit/                  # Audit logging
└── migrations/                 # SQL migration files
```
