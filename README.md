# byteChat

A terminal chat application built in Go. Users authenticate over **HTTPS**, then connect to a **TCP+TLS** messaging server for real-time chat. The client is a [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI.

## Requirements

- **Go 1.25+**
- A terminal that supports ANSI colors (Windows Terminal, iTerm2, etc.)

## Quick start

Open two terminals in the project root.

**Terminal 1 — server**

```powershell
# Create an admin account (first time only)
go run ./cmd/bytechat-server --create-admin admin:yourpassword

# Start the server
go run ./cmd/bytechat-server
```

**Terminal 2 — client**

```powershell
go run ./cmd/bytechat
```

On the welcome screen:

| Key | Action |
|-----|--------|
| `l` | Log in |
| `r` | Register a new account |
| `~` | Admin login |
| `q` / `Ctrl+C` | Quit |

There is no default admin account. You must run `--create-admin` at least once before admin login works.

## Server

```powershell
go run ./cmd/bytechat-server [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-https-addr` | `:8443` | HTTPS listen address (auth + admin API) |
| `-tcp-addr` | `:8444` | TCP+TLS listen address (real-time messaging) |
| `-create-admin` | *(empty)* | Create or promote an admin user (`username:password`) |

The server listens on:

- `https://localhost:8443` — registration, login, admin API
- `localhost:8444` — chat connections

TLS certificates are auto-generated on first run and stored under the data directory (see below). The client skips certificate verification for local development.

### Admin bootstrap

```powershell
go run ./cmd/bytechat-server --create-admin admin:yourpassword
```

This creates the user if it does not exist, or promotes an existing user to admin. Then log in from the client welcome screen with **`~`**.

Admin capabilities:

- View server stats and online users
- List and delete users (two-step confirm: `d` then `d` again)
- Toggle structured log categories
- Wipe the entire database (type `WIPE DATABASE` to confirm)

## Client

```powershell
go run ./cmd/bytechat [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-server` | `https://localhost:8443` | HTTPS auth / admin server URL |
| `-tcp` | `localhost:8444` | TCP+TLS chat server address |

Point these flags at a remote host if the server is not running locally.

### Chat controls

| Key | Action |
|-----|--------|
| `1` / `2` / `3` | Friends / Incoming / Outgoing tabs |
| `↑` / `↓` | Select contact or request |
| `a` | Add friend (opens modal) |
| `Enter` | Send message, accept incoming request, or submit modal |
| `Esc` | Close add-friend modal |
| `q` / `Ctrl+C` | Quit |

Messages and conversation history are loaded from the server. Only friends can message each other.

### Admin panel controls

After admin login (`~` on welcome screen):

| Key | Action |
|-----|--------|
| `1`–`4` | Dashboard / Users / Logs / Wipe tabs |
| `r` | Refresh current data |
| `↑` / `↓` | Select user or log category |
| `d` | Delete user (press twice to confirm) |
| `Space` | Toggle log category on/off |
| `Esc` | Back to welcome (cancels pending delete) |

## Data on disk

All persistent data lives under `~/.gochat/` (or `%USERPROFILE%\.gochat\` on Windows):

```
~/.gochat/
  server/
    gochat.db          SQLite database (users, messages, sessions, friends)
    cert.pem           TLS certificate (auto-generated)
    key.pem            TLS private key
    log_config.json    Log category toggles
  client/
    e2e_keys/          Friend public keys (E2E, when enabled)
    e2e_private.pem
    e2e_public.pem
```

Delete `gochat.db` to reset all server data, or use the admin **Wipe** tab.

## Logging

The server writes structured logs to stderr. Categories can be toggled from the admin **Logs** tab or by editing `log_config.json`:

| Category | What is logged |
|----------|----------------|
| `server` | Startup, listen addresses |
| `http` | HTTP method, path, status, duration |
| `tcp` | Client connect/disconnect (username only) |
| `messaging` | Message id, from, to (no message body) |
| `friends` | Friend requests and acceptances |
| `admin` | Admin actions (login, delete, wipe, log toggles) |
| `store` | Migrations, user create/delete, message store metadata |

**Never logged:** message body text, passwords, session tokens, or other auth secrets.

## Building binaries

```powershell
go build -o bytechat-server ./cmd/bytechat-server
go build -o bytechat-client ./cmd/bytechat
```

## Running tests

```powershell
go test ./...
```

## Architecture

```
client (TUI)
  ├─ HTTPS  →  router       register, login, admin API
  └─ TCP+TLS →  hub          authenticate with token, real-time messaging
                    ↓
                service      auth, messages, friends, history, admin
                    ↓
                store        database interface
                    ↓
                sqlite       SQLite implementation
```

## Project structure

```
cmd/
  bytechat/           Bubble Tea client
  bytechat-server/    HTTPS + TCP+TLS server

internal/
  client/             HTTP auth, admin API, and TCP chat clients
  logx/               Structured logging with category toggles
  paths/              ~/.gochat/ path helpers
  protocol/           Length-prefixed JSON frame codec
  router/             HTTP routes (auth + admin)
  server/             TCP hub, TLS, listener
  service/            Business logic
  store/              Store interface
    sqlite/           SQLite implementation and migrations
  tui/                Terminal UI screens and styles

pkg/types/            Shared types
```

## Wire protocol

Messages over TCP+TLS use a length-prefixed JSON envelope:

```
[4 bytes: payload length (uint32 big-endian)][N bytes: JSON payload]
```

Example envelope:

```json
{"type": 2, "data": {"to_username": "alice", "body": "hey"}}
```

The receiver unmarshals the envelope, switches on the type code, then unmarshals the inner payload into the appropriate struct. See `internal/protocol/` for message types and constants.

## Admin HTTP API

All admin routes except login require `Authorization: Bearer <token>` from admin login.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/admin/login` | Admin login |
| `GET` | `/api/admin/dashboard` | Stats, online users, log config |
| `GET` | `/api/admin/users` | List users |
| `DELETE` | `/api/admin/users/{username}` | Delete user |
| `POST` | `/api/admin/wipe` | Wipe DB (`{"confirm":"WIPE DATABASE"}`) |
| `GET` | `/api/admin/logs` | Get log config |
| `PUT` | `/api/admin/logs/{category}` | Toggle category (`{"enabled":true}`) |

Public auth routes: `POST /api/register`, `POST /api/login`.
