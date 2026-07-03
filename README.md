# Yaplane Live: Real-Time Forum & Messaging Platform

A full-stack community platform built with **Go, SQLite, and vanilla JavaScript**, combining public discussions with real-time private messaging in a responsive interface.

The project demonstrates HTTP API development, concurrent WebSocket connections, relational data modeling, and transactional persistence. One Go server serves the frontend, API, and live connections.

## Key features

- **Discussions:** categorized posts, replies, likes/dislikes, server-side search, filtering, sorting, and pagination.
- **Live updates:** posts, replies, reactions, and deletions update for members and guests.
- **Messaging:** private chat, typing indicators, presence, saved history, offline messages, and automatic reconnection.
- **Reliable delivery:** messages are acknowledged after storage; retry IDs prevent duplicate messages, and accounts support multiple connections.
- **Accounts and moderation:** bcrypt password hashing, expiring HttpOnly session cookies, owner-only profile details, reporting, and moderator actions with audit records.
- **Interface:** responsive layouts, light/dark themes, keyboard navigation, and loading/error states.

## Technology stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.27.1, `net/http`, `database/sql` |
| Database | SQLite, `mattn/go-sqlite3` (CGO) |
| Real-time communication | Gorilla WebSocket |
| Authentication | bcrypt, UUID session identifiers, HttpOnly cookies |
| Frontend | HTML, CSS, native JavaScript modules |
| Local environment | Docker and Docker Compose |

The frontend needs no framework, npm install, or build step. Database transactions, foreign-key constraints, input validation, and rate limiting support reliable application behavior.

## Run locally

Install Docker with Compose support. From the repository root:

```sh
docker compose up --build -d
```

Open **[http://localhost:8080](http://localhost:8080)** and register an account. The application creates its database automatically; a Docker volume preserves accounts and content across restarts.

```sh
docker compose down
```

This stops the app while keeping its data. Optional settings are listed in [.env.example](.env.example).

Without Docker, install Go 1.27.1 and a C compiler with CGO enabled, then run from the repository root:

```sh
go run -ldflags="-s -w" .
```

The native server also defaults to port **8080**.

## Optional demo content

For a populated demo, run these commands **instead of the first startup command above**, before a database exists in the Docker volume:

```sh
docker compose build
docker compose run --rm forum /app/seed-demo
docker compose up -d
```

This creates **5 fictional accounts, 15 posts, 30 replies, 90 reactions, and 18 messages**. The seed command refuses existing database files and never overwrites your data.

| ID | Username |
| --- | --- |
| 1 | `alex_demo` |
| 2 | `mia_demo` |
| 3 | `sam_demo` |
| 4 | `noor_demo` |
| 5 | `leo_demo` |

**Password for all demo accounts:** `DemoReview!2026`. Login uses the username, not the ID. These are shared member accounts; keep moderator credentials private.

To explore the demo, browse and react to discussions, then sign in as Alex and Mia in separate browser profiles/private contexts to try live chat and earlier message history.

For native Go, create a new demo file with `go run -ldflags="-s -w" ./cmd/seed-demo -database data/demo.db`, then set `DATABASE_PATH=data/demo.db` in your shell before starting the server.
