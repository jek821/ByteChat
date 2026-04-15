# byteChat

A CLI chat application using HTTPS for authentication and TCP+TLS for real-time messaging.

## Project Structure

```
internal/
  paths/
    paths.go            - Resolves filesystem paths for app data directories and the database file.

  protocol/
    protocol.go         - Frame reader/writer for the TCP connection. Defines the Envelope type,
                          message type constants, and the length-prefix framing format.

  router/
    router.go           - HTTP routes for registration and login. Returns a session token on
                          successful authentication.

  server/
    server.go           - Entry point for the TCP+TLS server. Wires together TLS config,
                          the listener, and the service layer.
    tls.go              - Loads or generates the TLS certificate and key, persists them to disk,
                          and builds the tls.Config.
    listener.go         - Accepts incoming TCP connections and spawns a goroutine per client
                          to handle the session lifecycle.

  service/
    service.go          - Business logic. Handles user creation (password hashing), authentication,
                          session token issuance, and message delivery.

  store/
    store.go            - The Store interface. Defines the database operations the rest of the
                          application depends on, decoupled from any specific implementation.
    sqlite/
      sqlite.go         - SQLite implementation of the Store interface. Raw SQL, no business logic.
      migrations.go     - Version-based schema migrations. Each version runs in a transaction and
                          is recorded in the schema_migrations table.
      migrations_test.go - Tests for migration correctness.
      sqlite_test.go    - Tests for the SQLite store implementation.

pkg/
  types/
    types.go            - Shared types used across packages (User, Session, Message).
```

## Architecture

```
client
  ├─ HTTPS  →  router      register, login, receive session token
  └─ TCP+TLS →  server     authenticate with token, send/receive messages in real time
                    ↓
                service     business logic
                    ↓
                store       database interface
                    ↓
                sqlite      SQLite implementation
```

## Wire Protocol

Messages are exchanged over TCP+TLS using a simple length-prefixed JSON format:

```
[4 bytes: payload length (uint32 big-endian)][N bytes: JSON payload]
```

The JSON payload is an envelope with a type code and a raw inner payload:

```json
{"type": 2, "data": {"to": 42, "body": "hey"}}
```

The receiver unmarshals the envelope first, switches on the type code, then unmarshals
the inner data into the appropriate struct.
