# boibritto — Project Structure & Architecture

**Status:** Planned / in migration
**Audience:** Team members onboarding to the codebase
**Last updated:** 2026-08-18

---

## 1. Summary

boibritto is being restructured from a single-package Go monolith (server-rendered HTML, cookie
sessions, Web Push) into a **JSON API backend + two Flutter clients** (Web and Android, one Flutter
codebase, two build targets).

Key decisions driving this restructure:

| Decision | What it replaces | Why |
|---|---|---|
| **Firebase Authentication** | Custom bcrypt + cookie sessions | Both clients need token-based auth (Android has no cookie jar the way a browser does); Firebase also gives us password reset, and custom claims for admin roles, for free |
| **Firebase Cloud Messaging (FCM)** | Web Push (VAPID) via `webpush-go` | One API for Android + Web push instead of two; free with no message limits; already paying the "add Firebase SDK" cost for Auth |
| **In-app messaging delivered via FCM, not polling** | `pollMessagesHandler` as the primary path | Real-time delivery without building our own persistent-connection infrastructure. Polling stays as a fallback/reconciliation call, not the main path |
| **Monorepo** (`backend/` + `app/` siblings) | N/A (new repo) | One person/small team owns both ends; atomic PRs keep API contract and Flutter models from drifting apart |
| **No server-rendered HTML** | `html/template`, `templates/*.html` | Both clients are Flutter; there is no browser rendering our templates anymore |

---

## 2. Repository layout (top level)

```
boibritto/
├── backend/            # Go API server
├── app/                # Flutter client (Web + Android targets)
├── contracts/          # OpenAPI spec — shared source of truth for the API shape
├── .github/workflows/  # CI, path-filtered so backend/app changes don't trigger each other's pipelines
└── README.md
```

**Why monorepo:** the biggest practical win is that a backend change and the corresponding Flutter
model/client change land in the *same commit*, so they can't silently go out of sync. The cost
(mixed toolchains in one repo) is handled by keeping `backend/` and `app/` as clean siblings with
their own dependency files (`go.mod` / `pubspec.yaml`) and CI jobs that only trigger on their own
path.

---

## 3. Backend structure (`backend/`)

```
backend/
├── cmd/
│   ├── api/
│   │   └── main.go                    # config → firebase client → db → services → router → server
│   └── migrate_users_to_firebase/
│       └── main.go                    # one-off: bulk-imports existing users into Firebase Auth
│
├── internal/
│   ├── config/
│   │   └── config.go                  # env: DATABASE_URL, FIREBASE_PROJECT_ID,
│   │                                   #      GOOGLE_APPLICATION_CREDENTIALS, CORS_ALLOWED_ORIGINS
│   │
│   ├── platform/
│   │   ├── postgres/
│   │   │   ├── postgres.go            # Connect() + goose migration runner
│   │   │   └── migrations/*.sql
│   │   ├── firebase/
│   │   │   └── firebase.go            # firebase.NewApp() → exposes *auth.Client + *messaging.Client
│   │   └── httpserver/
│   │       └── server.go              # *http.Server, timeouts, graceful shutdown
│   │
│   ├── apihttp/                       # cross-cutting HTTP concerns, no business logic
│   │   ├── router.go                  # mounts every domain's routes under /api/v1
│   │   ├── respond.go                 # respondJSON / respondError — one consistent JSON envelope
│   │   ├── cors.go                    # allows the Flutter Web origin
│   │   ├── errors.go                  # maps apperror sentinels → HTTP status codes
│   │   └── requestctx.go              # pulls authenticated userID / isAdmin out of context
│   │
│   ├── domain/
│   │   └── types.go                   # User{ID, FirebaseUID, Name, Email, WhatsAppNumber, ...}
│   │
│   ├── auth/
│   │   ├── middleware.go              # RequireAuth: verifies Firebase ID token, JIT-provisions user
│   │   ├── provision.go               # findOrCreateUser(firebaseUID, email, name)
│   │   └── store.go                   # getUserByFirebaseUID, insertUser, updateProfile
│   │
│   ├── profile/
│   │   ├── handler.go                 # GET/PATCH /api/v1/me
│   │   ├── service.go
│   │   └── dto.go
│   │
│   ├── books/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── store.go                   # DB queries
│   │   ├── covers.go                  # Open Library cover search integration
│   │   ├── cache.go                   # in-memory cover-search result cache
│   │   └── dto.go                     # JSON request/response shapes
│   │
│   ├── requests/                      # borrow requests
│   │   ├── handler.go
│   │   ├── service.go                 # includes ownership checks before accept/reject
│   │   ├── store.go
│   │   └── dto.go
│   │
│   ├── messages/
│   │   ├── handler.go                 # GET /threads, GET/POST /threads/{id}, poll (fallback), read
│   │   ├── service.go                 # SendMessage() persists, THEN calls push.Sender
│   │   ├── store.go
│   │   └── dto.go
│   │
│   ├── push/
│   │   ├── handler.go                 # POST /push/subscribe {platform, fcmToken}
│   │   ├── sender.go                  # FCM Sender interface + implementation
│   │   └── store.go                   # push_subscriptions table access
│   │
│   ├── admin/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── middleware.go              # RequireAdmin — checks a real Firebase custom claim
│   │
│   └── apperror/
│       └── errors.go                  # sentinel errors shared across all services
│
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── go.sum
```

### Design rules for this layer

- **Domain-first, not layer-first.** Code is grouped by what it's about (`books`, `messages`,
  `requests`) rather than by kind (all handlers together, all DB queries together). Working on
  messaging means working inside `internal/messages/` and nowhere else.
- **`handler.go` only knows HTTP.** Parses the request, calls `service.go`, writes the response.
  No SQL, no business rules.
- **`service.go` only knows business rules.** No `*http.Request`/`http.ResponseWriter` in sight.
  This is what makes services unit-testable without spinning up a real HTTP server.
- **`store.go` only knows SQL.** Thin data-access layer, no decision-making.
- **Cross-domain calls go through injected interfaces, not globals.** Example: `messages.Service`
  depends on a `push.Sender` interface (passed in at construction), not a global function — this
  lets tests substitute a fake sender instead of hitting real FCM.

---

## 4. Flutter structure (`app/`)

```
app/
├── lib/
│   ├── main.dart
│   ├── core/
│   │   ├── api/            # HTTP client, auth-header interceptor, error handling
│   │   ├── auth/            # FirebaseAuth wrapper, auth state stream, token refresh
│   │   ├── models/           # Dart classes mirroring backend DTOs (ideally OpenAPI-generated)
│   │   ├── push/              # FCM setup: token registration, foreground/background handlers
│   │   ├── theme/
│   │   └── router.dart         # shared navigation, used by both build targets
│   ├── features/                # mirrors backend's internal/ domains 1:1
│   │   ├── auth/                 # login/register screens
│   │   ├── books/                 # browse, detail, create, edit, my shelf
│   │   ├── requests/               # sent / incoming requests
│   │   ├── messages/                # thread list, thread detail
│   │   └── profile/
│   └── shared_widgets/
├── android/                        # google-services.json lives here
├── web/                             # Firebase web config lives here
├── test/
├── pubspec.yaml
└── firebase_options.dart            # generated once via `flutterfire configure`, shared by both targets
```

`features/` intentionally mirrors `backend/internal/`'s domain names — when in doubt about where a
piece of Flutter code goes, check what backend domain it talks to.

---

## 5. Where existing (pre-refactor) code goes

The current codebase is a flat, single-package Go app (`main.go`, `handlers.go`, `helpers.go`,
`middleware.go`, `cache.go`, `types.go`, `templates/`, `static/`). Nothing is deleted blindly —
this table is the map for the migration.

| Current file | New home | Action |
|---|---|---|
| `main.go` | `backend/cmd/api/main.go` | Rewrite — session middleware and template loading removed |
| `types.go` → `User` | `backend/internal/domain/types.go` | Move + edit: drop `PasswordHash`, add `FirebaseUID` |
| `types.go` → `pageData` and friends | split into each domain's `dto.go` | Delete `pageData` (HTML-only concept); redistribute `Book`, `BorrowRequest`, `ShelfPageData`, `ThreadData`, etc. into `books/dto.go`, `requests/dto.go`, `messages/dto.go` |
| `handlers.go`: login/register/logout/forget-password | — | **Delete.** Firebase Auth SDK replaces all of it client-side |
| `handlers.go`: book handlers | `backend/internal/books/handler.go` | Move, swap `ExecuteTemplate` for JSON responses, **add missing ownership checks** |
| `handlers.go`: `updaterequestStatusHandler` | `backend/internal/requests/handler.go` | Move, rename (typo fix), **add missing owner-authorization check** |
| `handlers.go`: messaging handlers | `backend/internal/messages/handler.go` | Move; `sendPushToUser` call becomes `push.Sender` call inside the service |
| `handlers.go`: admin handlers | `backend/internal/admin/handler.go` | Move, **replace fake "any logged-in user" check with real `RequireAdmin`** |
| `handlers.go`: push subscribe/unsubscribe | `backend/internal/push/handler.go` | Move + rewrite: payload changes from Web Push `{endpoint, keys}` to FCM `{platform, fcmToken}` |
| `handlers.go`: Open Library cover search | `backend/internal/books/covers.go` | Move as-is |
| `helpers.go`: `insertUser`, `authenticateUser`, `updatePassword` | `backend/internal/auth/store.go` | Rewrite — `authenticateUser`/`updatePassword` deleted (Firebase's job), `insertUser` signature changes |
| `helpers.go`: book queries | `backend/internal/books/store.go` | Move as-is |
| `helpers.go`: request queries | `backend/internal/requests/store.go` | Move as-is |
| `helpers.go`: message/thread queries | `backend/internal/messages/store.go` | Move as-is |
| `helpers.go`: `sendPushToUser`, `PushPayload` | `backend/internal/push/sender.go` | Rewrite — Web Push SDK call becomes FCM SDK call |
| `middleware.go`: `UnreadMessageCount` | — | **Delete.** Becomes a normal `GET /api/v1/me/unread-count` endpoint the client calls on demand, not middleware on every request |
| `cache.go` | `backend/internal/books/cache.go` | Move as-is |
| `db/db.go` | `backend/internal/platform/postgres/postgres.go` | Move as-is |
| `db/migrations/*.sql` | `backend/internal/platform/postgres/migrations/*.sql` | Move as-is, **plus new migrations**: add `users.firebase_uid`, later drop `users.password_hash`, change `push_subscriptions` shape for FCM tokens |
| `cmd/generate_vapid/` | — | **Delete.** No VAPID keys needed once FCM replaces Web Push |
| `templates/*.html`, `static/*`, `sw.js`, `manifest.json` | — | **Delete** (or archive outside the module). No server-rendered UI or PWA service worker with two Flutter clients |
| `Dockerfile` | `backend/Dockerfile` | Edit — remove `COPY` lines for templates/static/sw.js/manifest.json |
| `docker-compose.yml`, `.gitignore`, `dev`, `tmp/` | `backend/` (same names) | No changes needed |
| `go.mod` | `backend/go.mod` | Edit — remove `lib/pq`, `alexedwards/scs/*`, `webpush-go`, `bcrypt`; add `firebase.google.com/go/v4` |

---

## 6. New concepts this restructure introduces

- **`contracts/openapi.yaml`** — the single source of truth for the API's request/response shapes.
  Both the backend and the Flutter client should be built/generated against this spec so they
  can't silently drift.
- **`apperror` sentinel errors** — every service returns one of a small fixed set of errors
  (`ErrNotFound`, `ErrForbidden`, `ErrValidation`, `ErrConflict`, ...), mapped centrally to HTTP
  status codes. Every handler and every Flutter API-error-handler deals with one predictable shape.
- **Injected `push.Sender` interface** — `messages`, `requests`, and `books` services depend on
  this interface, not on FCM directly, so business logic can be unit-tested with a fake sender.

---

## 7. Migration order (build should stay green at every step)

1. Scaffold `backend/` + `app/`; move pure-data files first (`db.go`, `cache.go`, split `types.go`,
   `store.go` files, `covers.go`) — no auth dependency, compiles immediately after import fixes.
2. `internal/platform/firebase` + `internal/auth` — everything else depends on being able to
   identify a request's user.
3. `cmd/migrate_users_to_firebase` — run against existing data, confirm migrated users can log in
   before deleting `password_hash`.
4. `internal/books`, `internal/requests` handler + service layers, behind `auth.RequireAuth`, with
   ownership checks added.
5. `internal/push` (FCM sender) — needed before `messages`.
6. `internal/messages` — wired to call `push.Sender`.
7. `internal/admin` with real `RequireAdmin`.
8. Delete `templates/`, `static/`, `sw.js`, `manifest.json`, `cmd/generate_vapid`; prune `go.mod`.
9. Update `Dockerfile` last, once nothing it copies still exists.
10. Scaffold `app/` Flutter project (`flutter create`, `flutterfire configure`), build out
    `core/` first, then `features/` in the same order as the backend domains above.

---

## 8. Known open questions (not blocking, but flagged for later)

- **Read receipts / typing indicators**, if wanted eventually, need either a persistent connection
  (WebSocket) or much more frequent polling — current design (FCM push + occasional poll fallback)
  does not cover this.
- **Refresh-token / secure-storage strategy for Flutter Web** — browser storage is inherently less
  secure than Android's Keystore-backed secure storage; decide whether Web accepts that trade-off
  or needs a different token-handling approach.
- **WhatsApp Cloud API integration** — a genuine future candidate for a *true* server-to-server
  webhook (unlike in-app messaging, which correctly uses FCM push, not a webhook) if we ever want
  borrower/owner messaging to also reach WhatsApp directly. Out of scope for now.
