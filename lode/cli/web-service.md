# Web Service API

The clankers web service provides cloud storage, team collaboration, and cross-device synchronization via a Go API gateway.

## Architecture

```mermaid
flowchart TB
    subgraph Clients
        CLI[CLI Tool]
        Web[Web Dashboard]
    end
    
    subgraph Gateway["API Gateway (Go)"]
        Auth[Auth Middleware]
        Exchange[/auth/exchange]
        Sync[/sync/batch]
        Query[/query/*]
        JWKS[JWKS Endpoint]
    end
    
    subgraph Services
        UserSvc[User Service]
        SyncSvc[Sync Service]
        QuerySvc[Query Service]
    end
    
    subgraph Storage
        Postgres[(PostgreSQL)]
        Redis[(Redis)]
    end
    
    CLI -->|Bearer Token| Gateway
    Web -->|Bearer Token| Gateway
    
    Auth --> Exchange
    Auth --> Sync
    Auth --> Query
    
    Sync --> SyncSvc
    Query --> QuerySvc
    Exchange --> UserSvc
    
    UserSvc --> Postgres
    SyncSvc --> Postgres
    QuerySvc --> Postgres
    SyncSvc --> Redis
```

## Authentication

### POST /auth/exchange

Exchanges a WorkOS access token for internal API tokens.

**Request:**
```json
{
  "workos_token": "eyJ..."
}
```

**Response:**
```json
{
  "api_access_token": "eyJ...",
  "api_refresh_token": "eyJ...",
  "expires_in": 900,
  "user": {
    "id": "usr_123",
    "email": "user@example.com",
    "org_id": "org_456"
  }
}
```

### Token Claims

Internal JWT claims:

```json
{
  "sub": "usr_123",
  "org_id": "org_456",
  "email": "user@example.com",
  "scopes": ["sync:write", "query:read"],
  "iat": 1738230000,
  "exp": 1738230900
}
```

### POST /auth/refresh

Refresh an expiring access token.

**Request:**
```json
{
  "refresh_token": "eyJ..."
}
```

**Response:**
```json
{
  "api_access_token": "eyJ...",
  "api_refresh_token": "eyJ...",  // Rotated
  "expires_in": 900
}
```

### POST /auth/logout

Revoke refresh token server-side.

## Sync Endpoints

### POST /sync/batch

Upload a batch of sessions and messages.

**Headers:**
- `Authorization: Bearer <api_access_token>`
- `Content-Type: application/json`

**Request:**
```json
{
  "sessions": [
    {
      "id": "session-001",
      "title": "API Design",
      "project_path": "/home/user/project",
      "model": "claude-3-opus",
      "provider": "anthropic",
      "prompt_tokens": 1000,
      "completion_tokens": 500,
      "cost": 0.045,
      "created_at": 1738230000,
      "updated_at": 1738230100
    }
  ],
  "messages": [
    {
      "id": "msg-001",
      "session_id": "session-001",
      "role": "assistant",
      "text_content": "Here's the API design...",
      "model": "claude-3-opus",
      "prompt_tokens": 500,
      "completion_tokens": 200,
      "duration_ms": 1500,
      "created_at": 1738230000,
      "completed_at": 1738230100
    }
  ],
  "sync_timestamp": "2026-01-29T14:30:00Z"
}
```

**Response:**
```json
{
  "sessions_synced": 1,
  "messages_synced": 1,
  "sync_timestamp": "2026-01-29T14:35:00Z"
}
```

### GET /sync/status

Check sync status for user.

**Response:**
```json
{
  "last_sync": "2026-01-29T14:30:00Z",
  "total_sessions": 150,
  "total_messages": 1200
}
```

## Query Endpoints

### GET /query/sessions

List user's sessions with filtering.

**Query Parameters:**
- `limit`: Max results (default: 50, max: 200)
- `offset`: Pagination offset
- `search`: Search in title
- `from_date`: Filter by created_at >=
- `to_date`: Filter by created_at <=

**Response:**
```json
{
  "sessions": [...],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

### GET /query/sessions/:id

Get single session with messages.

**Response:**
```json
{
  "session": { ... },
  "messages": [...],
  "stats": {
    "message_count": 12,
    "total_tokens": 5000,
    "total_cost": 0.15
  }
}
```

## Multi-Tenancy

All endpoints enforce `org_id` isolation:

1. JWT contains `org_id` claim
2. Middleware extracts org_id to context
3. Queries filter by `WHERE org_id = $1`
4. Users cannot access other orgs' data

## Rate Limiting

| Endpoint | Limit |
|----------|-------|
| /auth/* | 10 req/min |
| /sync/batch | 60 req/min |
| /query/* | 120 req/min |

Rate limit headers:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

## Error Responses

Standard error format:

```json
{
  "error": {
    "code": "invalid_token",
    "message": "The provided token has expired",
    "details": {}
  }
}
```

Common codes:
- `unauthorized` - Invalid or missing token
- `forbidden` - Valid token but insufficient permissions
- `rate_limited` - Too many requests
- `invalid_request` - Malformed request
- `conflict` - Resource conflict
- `internal_error` - Server error

Links: [sync](sync.md), [auth](auth.md), [cli architecture](architecture.md)
