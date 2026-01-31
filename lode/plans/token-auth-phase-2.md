# Token Authentication (Phase 2)

Implementation plan for static token authentication - simple API key style for small teams and shared servers.

## Overview

Adds `auth=token` mode where a static token is sent with each request. Token stored in plain text in config file.

**Priority**: Medium (after Phase 1 is stable)
**Estimated Time**: 1-2 days

## CLI Changes

### Config Extensions

```json
{
  "profiles": {
    "team": {
      "endpoint": "https://team-server.com",
      "auth": "token",
      "token": "sk_live_abc123..."
    }
  }
}
```

New config commands:
```bash
clankers config set auth token
clankers config set token sk_live_abc123...
```

### Sync Client Changes

**File**: `packages/cli/internal/sync/client.go`

```go
func (c *Client) sendBatch(data SyncData) error {
    req, _ := http.NewRequest("POST", c.endpoint+"/sync/batch", body)
    
    if c.authMode == "token" {
        req.Header.Set("Authorization", "Bearer "+c.token)
    }
    
    return c.httpClient.Do(req)
}
```

## Web Service Changes

### Token Validation Middleware

**File**: `apps/web-service/internal/middleware/auth.go`

```go
func TokenAuth(db *db.Manager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract Bearer token
            // Map token to tenant database
            // Add tenant_id to context
            next.ServeHTTP(w, r)
        })
    }
}
```

### Token-to-Tenant Mapping

Options:
1. **Hash-based**: Token hash = database name (simple, no DB lookup)
2. **Lookup table**: `tokens` table maps token hash → tenant_id

Recommended: Hash-based for simplicity

```go
func tokenToTenant(token string) string {
    // Use first 16 chars of SHA256 hash as tenant name
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:8])
}
```

### Database Provisioning

When first request arrives with new token:
1. Check if tenant DB exists
2. If not, create new Turso database
3. Run migrations
4. Store token hash → tenant mapping (optional)

## Security Considerations

- Token stored in **plain text** in config (acceptable for Phase 2)
- Token transmitted over HTTPS (required)
- Token can be revoked by deleting from web service
- No keyring storage yet (comes in Phase 3)

## Migration Path

Existing no-auth users:
```bash
# Current (Phase 1)
clankers config set endpoint https://server.com
clankers config set auth none

# Upgrade to token auth
clankers config set auth token
clankers config set token sk_live_...
```

## Testing

```bash
# CLI side
$ clankers config set auth token
$ clankers config set token test-token-123
$ clankers sync now

# Web service
$ curl -H "Authorization: Bearer test-token-123" \
  https://server.com/sync/batch \
  -d '{"sessions":[],"messages":[]}'
```

## Dependencies

- None beyond existing HTTP client
- Optional: Add token hashing (crypto/sha256 is stdlib)

Links: [Phase 1 CLI](../cli/architecture.md), [Web Service](../web-service/overview.md)
