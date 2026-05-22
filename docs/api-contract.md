# REST API Contract — Gateway

All routes go through the API gateway at port **8080**.  
JWT must be present in `Authorization: Bearer <token>` for every route **except** `/health`.

---

## Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/auth/token` | ✅ | Issue a JWT. In dev mode no real auth provider is used. |
| `POST` | `/api/v1/submissions` | ✅ | Upload a contestant binary → routed to the **submission** service. |
| `GET` | `/api/v1/submissions/:id` | ✅ | Get the status of a previously uploaded submission. |
| `GET` | `/api/v1/runs/:id` | ✅ | Get run status and associated performance metrics. |
| `GET` | `/api/v1/leaderboard` | ✅ | Current top-50 leaderboard (REST fallback). |
| `WS` | `/ws/leaderboard` | ✅ | WebSocket stream proxied to the **leaderboard** service. |
| `GET` | `/health` | ❌ | Kubernetes liveness / readiness probe. |

---

## Authentication

```
Authorization: Bearer <token>
```

Tokens are obtained via `POST /api/v1/auth/token`.  
In **dev mode** the gateway issues self-signed JWTs with no upstream identity provider — useful for local testing.

---

## Downstream Service Mapping

| Route prefix | Target service |
|---|---|
| `/api/v1/auth/*` | gateway (self-handled) |
| `/api/v1/submissions` | submission service |
| `/api/v1/runs` | ingester / scorer service |
| `/api/v1/leaderboard` | leaderboard service |
| `/ws/leaderboard` | leaderboard service |

---

## Request / Response Examples

### POST /api/v1/auth/token

**Request**

```json
{
  "contestant_id": "abc-123"
}
```

**Response — 200 OK**

```json
{
  "token": "<jwt>",
  "expires_in": 3600
}
```

---

### POST /api/v1/submissions

**Request** — `multipart/form-data`

| Field | Type | Description |
|---|---|---|
| `binary` | file | Compiled contestant binary |
| `language` | string | `"cpp"` \| `"rust"` \| `"go"` |

**Response — 202 Accepted**

```json
{
  "submission_id": "sub-456",
  "status": "queued"
}
```

---

### GET /api/v1/submissions/:id

**Response — 200 OK**

```json
{
  "submission_id": "sub-456",
  "status": "running",
  "run_id": "run-789"
}
```

---

### GET /api/v1/runs/:id

**Response — 200 OK**

```json
{
  "run_id": "run-789",
  "status": "completed",
  "metrics": {
    "p50_ms": 1.23,
    "p90_ms": 2.45,
    "p99_ms": 5.67,
    "max_tps": 120000,
    "correctness": 0.998
  }
}
```

---

### GET /api/v1/leaderboard

**Response — 200 OK**

```json
{
  "entries": [
    {
      "rank": 1,
      "contestant_id": "abc-123",
      "composite": 98.7
    }
  ]
}
```

---

### GET /health

**Response — 200 OK**

```json
{ "status": "ok" }
```
