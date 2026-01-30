# CLI Authentication

Authentication is **optional** and will be implemented in phases:

## Phase 1: No Authentication (Current)

For personal deployments or trusted networks:

```bash
# Configure endpoint only
clankers config set endpoint https://my-clankers-server.com

# Sync runs without any auth
clankers sync status
```

## Phase 2: Static Token (Future)

Simple API key authentication (planned):

```bash
# Configure endpoint with static token
clankers config set endpoint https://my-clankers-server.com
clankers config set auth token
clankers config set token my-api-key-123
```

**Storage**: Token will be stored in plain text in config file for Phase 2. Keyring integration planned for Phase 3.

## Phase 3: WorkOS AuthKit (Future)

Full SSO with WorkOS (planned):

```bash
# Login via WorkOS device code flow
clankers login
# Configures endpoint, tokens, and profile automatically
```

**Storage**: WorkOS tokens will use OS keyring (secure) once implemented in Phase 3.

## Configuration

### Phase 1 (No Auth)

```bash
# Set endpoint
clankers config set endpoint https://my-clankers-server.com

# View config
clankers config list
```

### Config File Location

Stored at platform-appropriate location:
- macOS: `~/Library/Application Support/clankers/config.json`
- Linux: `~/.config/clankers/config.json` (XDG)
- Windows: `%APPDATA%/clankers/config.json`

### Config File Format

```json
{
  "profiles": {
    "default": {
      "endpoint": "https://my-server.com",
      "auth": "none"
    },
    "work": {
      "endpoint": "https://work-server.com",
      "auth": "none"
    }
  },
  "active_profile": "default"
}
```

## Profile Management

Profile creation and deletion is done through the **web interface**. CLI only switches between existing profiles:

```bash
# List available profiles
clankers config profiles list

# Switch to a profile
clankers config profiles use work

# Current profile shown in status
clankers sync status
```

## Environment Variables

| Variable | Purpose | Override Priority |
|----------|---------|-------------------|
| `CLANKERS_ENDPOINT` | Sync endpoint URL | Highest |
| `CLANKERS_PROFILE` | Active profile | Medium |
| `CLANKERS_SYNC_ENABLED` | Master sync toggle | High |

## Roadmap

| Phase | Feature | Status |
|-------|---------|--------|
| 1 | No-auth mode, plain text config | Current |
| 2 | Static token auth, plain text storage | Planned |
| 3 | WorkOS auth, keyring storage | Planned |

Links: [cli architecture](architecture.md), [sync](sync.md)
