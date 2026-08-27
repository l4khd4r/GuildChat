# GuildChat API endpoints

All routes are served by `internal/router/router.go`. The base URL is
`http://localhost:$PORT` (`PORT` comes from the environment; the Postman
collection defaults to `8080`).

Requests and responses are JSON. Protected routes require a bearer token
obtained from `POST /auth/login`:

```
Authorization: Bearer <token>
```

A protected route answers `401` when the header is missing, is not in
`Bearer <token>` form, or carries a token that fails validation.

---

## Public

### `GET /`

Welcome message.

`200` → `{ "message": "Welcome to My World!" }`

### `GET /health`

Liveness check.

`200` → `{ "status": "healthy" }`

### `POST /users`

Register a user.

| Field | Rules |
| --- | --- |
| `username` | required, 3–20 chars |
| `email` | required, valid email, ≤255 chars |
| `password` | required, ≥8 chars |

```json
{ "username": "guildmaster", "email": "gm@example.com", "password": "ChangeMe123!" }
```

| Code | Meaning |
| --- | --- |
| `201` | created, returns the user |
| `400` | payload failed validation |
| `409` | username or email already taken |
| `500` | anything else |

Success body (this shape is `dto.UserResponse`, reused by every endpoint below
that returns a user):

```json
{
  "id": 1,
  "username": "guildmaster",
  "email": "gm@example.com",
  "created_at": "2026-08-27T10:00:00Z",
  "updated_at": "2026-08-27T10:00:00Z"
}
```

The password hash is never part of a response.

### `GET /users/:id`

Public profile of any user.

| Code | Meaning |
| --- | --- |
| `200` | returns the user |
| `400` | `:id` is not a number |
| `404` | no such user |

### `POST /auth/login`

Exchange credentials for a JWT.

```json
{ "email": "gm@example.com", "password": "ChangeMe123!" }
```

| Code | Meaning |
| --- | --- |
| `200` | returns the token and a short user summary |
| `400` | payload failed validation |
| `401` | invalid credentials |
| `500` | anything else |

```json
{
  "token": "eyJhbGciOi...",
  "user": { "id": 1, "username": "guildmaster", "email": "gm@example.com" }
}
```

Note this `user` object is trimmed (no timestamps) and is not
`dto.UserResponse`.

---

## Account — protected

### `GET /me`

The authenticated user. `200` → user. `404` if the account no longer exists.

### `PUT /me`

Update the authenticated user. **Both fields are required** — this is a full
replace, not a patch, so sending only one clears nothing but fails validation.

```json
{ "username": "newname", "email": "new@example.com" }
```

| Code | Meaning |
| --- | --- |
| `200` | returns the updated user |
| `400` | payload failed validation |
| `404` | account no longer exists |
| `409` | username or email already taken |

### `DELETE /me`

Delete the authenticated user.

`204` with no body on success, `404` if the account is already gone.

Friendship rows referencing the user are removed by
`ON DELETE CASCADE` on both foreign keys.

---

## Friendships — protected

A friendship is a single row holding `requester_id`, `receiver_id` and a
`status` of `pending` / `accepted` / `rejected`. A unique index on the
*unordered pair* means two users can only ever have one friendship row between
them, whichever of them asked first.

Watch the two meanings of `:id` here:

- **sending** is addressed by **user** id — you know who you want to befriend,
  and the friendship does not exist yet;
- **accepting and rejecting** are addressed by **friendship** id — the row
  already exists, and the list endpoints hand you its id.

Responses use two shapes:

`FriendshipResponse`, returned by the three write endpoints:

```json
{
  "id": 42,
  "requester_id": 1,
  "receiver_id": 2,
  "status": "pending",
  "created_at": "2026-08-27T10:00:00Z",
  "updated_at": "2026-08-27T10:00:00Z"
}
```

`FriendRequestResponse`, returned by the two list endpoints. `id` is the
**friendship** id (POST it straight to accept or reject), `created_at` is when
the request was sent, and `user` is whoever is on the other side:

```json
{
  "id": 42,
  "user": { "id": 2, "username": "rogue", "email": "rogue@example.com",
            "created_at": "...", "updated_at": "..." },
  "status": "pending",
  "created_at": "2026-08-27T10:00:00Z"
}
```

### `POST /users/:id/friend-request`

Send a request to user `:id`. The requester is the caller.

| Code | Meaning |
| --- | --- |
| `201` | created, status `pending` |
| `400` | `:id` is not a number, or you addressed yourself |
| `409` | you two are already linked, in either direction and any status |
| `500` | anything else |

`409` also covers two users sending to each other simultaneously: the loser of
the race hits the unique index and is reported as an existing friendship rather
than a server error.

### `POST /friend-request/:id/accept`

Accept a pending request addressed to you. `:id` is the friendship.

| Code | Meaning |
| --- | --- |
| `200` | returns the friendship, now `accepted` |
| `400` | `:id` is not a number |
| `404` | no such request, you are not its receiver, or it is not pending |

The three `404` cases are deliberately indistinguishable. Authorisation is
enforced in the SQL — the `UPDATE` matches on `receiver_id = <caller>` — so you
cannot accept a request addressed to someone else, and separating the cases
would leak the existence of other users' requests.

### `POST /friend-request/:id/reject`

Turn down a pending request addressed to you. Same shape and same rule as
accept.

`200` returns the friendship with status `rejected`. Note the row is **deleted**
rather than marked, so the body describes something that no longer exists. That
is deliberate: the unique index covers the pair whatever its status, so a
leftover rejected row would lock those two users out of ever requesting again.

### `GET /me/friends`

Users the caller has an accepted friendship with, in either direction — it does
not matter who originally sent the request.

`200` → array of users (`[]` when there are none, never `null`), newest
friendship first.

### `GET /me/friend-requests`

Inbox: pending requests **received** by the caller. `user` is the sender.

`200` → array of `FriendRequestResponse`, newest first.

### `GET /me/friend-requests/sent`

Outbox: pending requests **sent** by the caller. `user` is the receiver.

`200` → array of `FriendRequestResponse`, newest first.

A request that disappears from this list was either accepted (the user now
shows up in `/me/friends`) or rejected — rejected rows are deleted, so they
never linger here.

---

## Testing

`postman/collections/GuildChat API` holds a runnable collection covering every
route above, including the failure cases (`400`, `401`, `404`, `409`). It runs
top to bottom against a fresh database: it registers three throwaway users,
logs them in, exercises the full request → accept and request → reject → resend
flows, then deletes the accounts it made.
