# GuildChat — Build & Run Diagnostics (2026-08-21)

**Verdict: the app is NOT running, and it cannot run — `go build ./...` fails.**
Nothing is listening on `:8080` (the port in `app/.env`), and no server process exists.
Postgres *is* up and healthy.

No code was changed while producing this report.

---

## What is running

| Component | State | Evidence |
|---|---|---|
| Postgres (`guildchat-postgres`, postgres:13) | **Up** ~2h, `0.0.0.0:5432->5432` | `docker ps` |
| `users` table | **Exists**, with `id, username, email, password_hash, created_at, updated_at` | `\d users` |
| Go API server | **Not running** | nothing listening on `:8080`; no matching process |

---

## Blocking problem (compile error)

`app/internal/service/user_service.go:26`

```go
return s.userRepo.CreateUser(
    ctx, username, email, passwordHash,
), nil
```

```
internal/service/user_service.go:26:9: multiple-value s.userRepo.CreateUser(...)
    (value of type (*repository.User, error)) in single-value context
```

`repository.CreateUser` returns **two** values (`*User, error`), but the call is being
used as a single value and then paired with a literal `nil`. Go does not allow a
multi-value call to be spread into a larger expression list.

Because `service` fails to compile, every package that imports it (`handler`, `router`,
`cmd/server`) fails too — that is why the build output only shows this one error.

## Second problem (hidden behind the first)

`app/internal/handler/user_handler.go:22-24` — **syntax error**: struct fields have
trailing commas after the struct tags.

```go
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`,      // <- comma
	Email    string `json:"email" binding:"required,email"` ,  // <- comma
	Password string `json:"password" binding:"required"`,      // <- comma
}
```

Struct fields are newline-separated, not comma-separated. Confirmed with `gofmt -e`:

```
internal/handler/user_handler.go:22:54: expected ';', found ','
internal/handler/user_handler.go:23:58: expected ';', found ','
internal/handler/user_handler.go:24:54: expected ';', found ','
```

This will surface as the next error as soon as the service error is fixed.

---

## Non-blocking issues found

1. **`.env` typo** — `db_SSLMODE=disable` should be `DB_SSLMODE=disable`.
   Env var lookup is case-sensitive, so `cfg.Database.SSLMode` is empty and the DSN
   ends with `?sslmode=`. Verified this still connects against the local container
   (pgx treats empty as `prefer`), so it is latent, not the current failure — but it
   silently disables the intended explicit `disable` mode.

2. **`User` struct is missing timestamp fields.** `repository.User` has only
   `ID, Username, Email, PasswordHash`, while the table has `created_at` / `updated_at`.
   The working-tree diff removed the `&user.createdAt` / `&user.updatedAt` scan targets
   (they were unexported and not declared on the struct anyway). Functionally consistent
   right now — the `INSERT ... RETURNING` list matches the four `Scan` targets — just
   noting the API will never return timestamps.

3. **`app/Dockerfile` is empty (0 bytes)** and `app/.env.example` is empty, so the
   compose setup covers Postgres only; the API has to be run by hand.

4. **Style, not errors** — `type Config struct` / `type User struct` put the opening
   brace on the next line in `database/postgres.go` and `repository/user_repository.go`.
   This compiles (no semicolon is inserted after `struct`), but `gofmt` will rewrite it.

---

## How this was checked

```bash
cd app
go build ./...                     # compile
go vet ./...                       # same error
gofmt -e internal/handler/user_handler.go   # exposed the masked syntax error
ss -ltnp | grep 8080               # nothing listening
docker ps                          # postgres up
docker exec guildchat-postgres psql -U postgres -d guildchat -c '\d users'
```

## To get it running (after the two compile errors are fixed)

```bash
cd app
go run ./cmd/server
curl localhost:8080/health
curl -X POST localhost:8080/users \
  -H 'content-type: application/json' \
  -d '{"username":"a","email":"a@b.c","password":"secret"}'
```

Note: `POST /users` currently stores the raw password in `password_hash` — the handler
passes `req.Password` straight through the service to the repository, with no hashing.
