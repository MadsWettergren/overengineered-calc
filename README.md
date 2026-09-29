# Over-engineered Calculator

A small, production-shaped REST API for performing basic arithmetic and
keeping a history of past calculations, written in Go.

The name is the assignment's joke, not mine: the calculator itself is
trivial. What this project actually demonstrates is the structure,
testing, and tooling around it — the kind of decisions that matter once a
service has to keep running for years, not just work once in a demo.

## Go experience note

This is my first significant Go project; I have production experience in
C# and .NET and leaned on that for the overall design.

## Architecture

```text
HTTP request
    │
    ▼
internal/httpapi      (routing, JSON, status codes, auth middleware)
    │
    ├──────────────► internal/auth   (accounts, bcrypt, tokens)
    ▼
internal/application   (use-case orchestration)
    │
    ▼
internal/calculator    (arithmetic rules — no HTTP, no storage)
    │
    ▼
internal/history       (storage interface + in-memory implementation)
```

Each layer only knows about the layer directly below it, and only through
an interface where one is needed:

| Package | Responsibility | Knows about HTTP? | Knows about storage? |
|---|---|---|---|
| `internal/calculator` | Arithmetic rules | No | No |
| `internal/history` | Calculation records and storage | No | — |
| `internal/application` | Coordinates calculator + history | No | Only via `history.Repository` |
| `internal/httpapi` | Decoding, routing, status codes | Yes | No |
| `cmd/api` | Wires everything together | Yes (constructs the server) | Yes (constructs the repository) |

This separation is what makes `internal/calculator` and
`internal/application` testable with plain unit tests, with no HTTP server
or database involved — and what would let a PostgreSQL-backed
`history.Repository` replace the in-memory one later without touching the
calculator or the HTTP layer at all.

History is currently stored in memory and is lost when the process
restarts. That's a deliberate scope decision, not an oversight — see
[Design decisions](#design-decisions).

## Running it

### With Docker Compose (recommended)

```bash
docker compose up --build
```

The API is now listening on `http://localhost:8080`.

Stop it with `Ctrl+C`, or from another terminal:

```bash
docker compose down
```
### Web client

Open `http://localhost:8080/` in a browser for a small page where you can
register, log in, run calculations, and view history. It's a single static
HTML file embedded in the binary and served by the API itself, so there's
nothing extra to build or run.

### With Go directly

Requires Go 1.27 or later.

```bash
go run ./cmd/api
```

## Using the API

### Authenticate

Register an account:

```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct horse battery"}'
```

Log in to get a bearer token:

```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct horse battery"}'
```

```json
{ "token": "3f9a1c..." }
```

Use the token on every calculator request:

```bash
curl -X POST http://localhost:8080/v1/calculations \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer 3f9a1c...' \
  -d '{"operation":"multiply","left":6,"right":7}'
```

Missing or invalid tokens get a `401`.

### Perform a calculation

```bash
curl -X POST http://localhost:8080/v1/calculations \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer 3f9a1c...' \
  -d '{"operation":"multiply","left":6,"right":7}'
```

```json
{
  "id": "b3f1...",
  "operation": "multiply",
  "left": 6,
  "right": 7,
  "result": 42,
  "created_at": "2026-09-23T10:55:06Z"
}
```

Supported operations: `add`, `subtract`, `multiply`, `divide`.

### List calculation history

```bash
curl http://localhost:8080/v1/calculations \
  -H 'Authorization: Bearer 3f9a1c...'
```

### Check liveness

```bash
curl http://localhost:8080/health/live
```

### API documentation

The full OpenAPI 3.0 specification is served by the running service itself:

```bash
curl http://localhost:8080/openapi.yaml
```

Paste that output into [editor.swagger.io](https://editor.swagger.io) for
an interactive view you can send requests from directly, without installing
anything locally.

### Error format

Errors share one shape across the API:

```json
{
  "error": {
    "code": "division_by_zero",
    "message": "cannot divide by zero"
  }
}
```

| Status | Meaning |
|---|---|
| `400` | Malformed JSON, unknown fields, or an unsupported operation |
| `401` | Missing or invalid bearer token |
| `422` | Well-formed request that isn't a valid calculation (e.g. division by zero) |
| `404` | Unknown path |
| `405` | HTTP method not supported on that path |

## Testing

```bash
go test -race ./...
```

With coverage:

```bash
go test ./... -cover
```

Tests are colocated with the code they cover (`*_test.go` next to the
package it tests), which is Go convention. `internal/calculator` and
`internal/history` are tested in isolation with no HTTP server involved;
`internal/httpapi` is tested by driving the `Handler` directly through
`net/http/httptest`, which is fast and needs no real network socket.

## Design decisions

**In-memory storage instead of PostgreSQL.** The assignment lists
persistence-grade storage as a nice-to-have, not a must-have. The
`history.Repository` interface is the seam a PostgreSQL implementation
would plug into — `internal/application` and `internal/httpapi` depend
only on that interface, so adding a database-backed repository later
would not require changing either of them.

**Structured requests instead of a free-text expression parser.**
`{"operation": "add", "left": 2, "right": 3}` avoids the ambiguity of
parsing arbitrary strings like `"2 + 3 * 4"` — operator precedence,
parentheses, and malformed input all become non-issues. It's a
deliberately narrower scope than a "real" calculator, in exchange for a
smaller and more predictable surface area.

**`float64` instead of a decimal type.** For addition, subtraction,
multiplication, and division of arbitrary numbers, some floating-point
imprecision (e.g. `0.1 + 0.2`) is an accepted trade-off here in exchange
for not adding an external decimal library. A production system handling
money would make the opposite trade-off.

**No web framework.** The API surface is three routes. Go's standard
`net/http` handles that without needing to evaluate, pin, and maintain a
router dependency.

**A single container, not microservices.** The domain is small enough
that splitting it into separate services would add network calls,
deployment complexity, and failure modes without adding real isolation
benefits. The "best-effort microservice architecture" nice-to-have is
addressed instead by keeping the *internal* boundaries clean — the
layering above is what would let pieces be extracted into separate
services later, if the domain ever grew enough to justify it.

**Opaque in-memory tokens instead of JWTs.** A single-instance service has
no need for a token format that's independently verifiable without a
shared store — that's the problem JWTs solve for multi-instance
deployments. The trade-off: tokens don't survive a restart and never
expire. A production version would add expiry at minimum, and JWTs (or a
shared token store) the moment there's more than one instance of this
service running.

**History is shared across accounts, not scoped per user.** Auth here
gates *access* to the calculator rather than partitioning data by who's
using it — every authenticated user sees the same history. Scoping
history to individual users is a natural next step (it would mean
associating each `history.Calculation` with a user ID), but it's a change
to `internal/history` and `internal/application`, not just the HTTP
layer, so it's being called out here rather than folded in silently.

**Scope deliberately left out.** No CI pipeline and no public deployment.
The service runs identically anywhere Docker does (`docker compose up
--build`), and the test suite is one command (`go test -race ./...`), so
both would be straightforward to add. I prioritized the parts that change
the code and its design (authentication, the web client) over the parts
that change how it is hosted.

## Project layout

```text
cmd/api/                   Application entry point (dependency wiring)
internal/calculator/       Arithmetic rules
internal/history/          Calculation records + storage interface
internal/auth/             Accounts, password hashing, token handling
internal/application/      Use-case orchestration
internal/httpapi/          HTTP handlers, middleware, OpenAPI spec, web client
Dockerfile                 Multi-stage build (compile, then minimal runtime image)
docker-compose.yml         Single-command run
```