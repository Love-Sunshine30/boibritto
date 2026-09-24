# Boibritto

Boibritto is a peer-to-peer book-sharing platform for students. Users can list books they own, discover books shared by others, request to borrow them, confirm handoffs, and communicate through request-based message threads.

## Current backend

The repository currently contains the Go backend.

### Stack

- Go 1.26
- Chi HTTP router
- PostgreSQL
- Firebase Authentication
- Firebase Cloud Messaging
- Firebase Firestore for message threads
- Goose for database migrations
- Docker Compose for local PostgreSQL

## API

The API is versioned under:

```text
/api/v1
```

Authenticated endpoints use a Firebase ID token:

```http
Authorization: Bearer <firebase-id-token>
```

The API returns a common JSON envelope.

Successful responses look like:

```json
{
  "data": {}
}
```

Errors look like:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "..."
  }
}
```

The OpenAPI contract is in `contracts.yml`.

### Main endpoints

| Area | Endpoints |
| --- | --- |
| Health | `GET /healthz` |
| Books | `GET/POST /api/v1/books` |
| Book | `GET/PATCH/DELETE /api/v1/books/{id}` |
| Borrow requests | `POST /books/{id}/requests`, `GET /requests/sent`, `GET /requests/incoming` |
| Request state | `PATCH /requests/{id}`, `POST /requests/{id}/confirm`, `POST /requests/{id}/return` |
| Profile | `GET/PATCH /me`, `GET /users/{id}` |
| Messages | `GET /threads`, `POST /threads/{id}/messages` |
| Push notifications | `POST /push/subscribe`, `POST /push/unsubscribe` |
| Book forum | `GET/POST /books/{id}/forum` |
| Forum post | `PATCH/DELETE /forum/{postID}` |
| Admin | `PATCH /admin/books/{id}/cover` |

## Project structure

```text
backend/
├── cmd/
│   ├── api/
│   └── migrate_users_to_firebase/
├── internal/
│   ├── admin/
│   ├── apihttp/
│   ├── app/
│   ├── apperror/
│   ├── auth/
│   ├── books/
│   ├── config/
│   ├── domain/
│   ├── forum/
│   ├── messages/
│   ├── platform/
│   │   ├── firebase/
│   │   ├── httpserver/
│   │   ├── logging/
│   │   └── postgres/
│   ├── profile/
│   ├── push/
│   └── requests/
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── .env.dev.example
```

Each domain owns its handlers, DTOs, services, and stores. The HTTP router composes these domains under `/api/v1`.

## Local development

### 1. Start PostgreSQL

From `backend/`:

```bash
docker compose up -d postgres
```

PostgreSQL is exposed on port `5073`.

### 2. Configure environment variables

Copy the example environment file:

```bash
cp .env.dev.example .env.dev
```

Set the Firebase project and service-account configuration required by the backend.

The example configuration uses:

```text
PORT=8080
DATABASE_URL=postgres://boibritto_user:ofmiceandmen@localhost:5073/boibritto?sslmode=disable
```

### 3. Run the API

```bash
go run ./cmd/api
```

The server starts on the configured port. Database migrations are applied automatically during startup.

### 4. Check the server

```bash
curl http://127.0.0.1:8080/healthz
```

A healthy response has the form:

```json
{
  "data": {
    "status": "ok",
    "database": "ok"
  }
}
```

## Borrowing flow

The main borrowing workflow is:

1. A student lists a book.
2. Another student sends a borrow request.
3. The owner accepts or rejects the request.
4. When accepted, a message thread is created for the request.
5. Both participants confirm the handoff.
6. After both confirmations, the request becomes `active` and the book becomes unavailable.
7. The owner marks the book as returned.
8. The request becomes `returned` and the book becomes available again.

Request statuses are:

```text
pending
accepted
active
rejected
returned
```

## Authentication

Firebase Authentication is used for API authentication.

The backend verifies the Firebase ID token, finds or creates the corresponding local user, and places the local user in the request context.

Admin endpoints additionally require the local user's admin flag.

## Messaging

Messages are associated with borrow requests. Accepting a request creates a thread for the owner and requester.

Messages are stored through Firebase Firestore, while user and book information remains in PostgreSQL.

Firebase Cloud Messaging is used for push notifications.

## Database migrations

PostgreSQL migrations are stored under:

```text
internal/platform/postgres/migrations/
```

The API applies pending migrations during startup.

## Testing

Run the Go test suite from `backend/`:

```bash
go test ./...
```

## Notes

The API contract should be kept synchronized with the actual handlers and DTOs in `internal/`. `contracts.yml` documents the currently implemented HTTP surface rather than the older planned architecture.
