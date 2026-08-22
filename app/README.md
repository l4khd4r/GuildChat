# GuildChat


That's the structure I'd recommend for GuildChat.
```
                         Client
                       ↙        ↘
              Request DTO      Response DTO
                   ↓                ↑
                   Handler
                     ↓              ↑
                   Service
                     ↓              ↑
                   Model
                     ↓
                Repository
                     ↓
                 PostgreSQL
```
have successfully reached this flow
```
POST /users
    ↓
JSON → CreateUserRequest
    ↓
Gin binding + validation
    ↓
UserService
    ↓
HashPassword()
    ↓
UserRepository
    ↓
PostgreSQL
    ↓
model.User
    ↓
UserResponse
    ↓
JSON response
```



##### example of bad request

```
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/users \
                                          -H "Content-Type: application/json" \
                                          -d '{
                                        "username": "ab",
                                        "email": "lakhdar",
                                        "password": "123"
                                      }'
{"error":"Key: 'CreateUserRequest.Username' Error:Field validation for 'Username' failed on the 'min' tag\nKey: 'CreateUserRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag\nKey: 'CreateUserRequest.Password' Error:Field validation for 'Password' failed on the 'min' tag"}⏎
prismo@fedora ~/D/GuildChat (main)>
```


#### example of 201 request
```
validation for 'Email' failed on the 'email' tag\nKey: 'CreateUserRequest.Password' Error:Field validation for 'Password' failed on the 'min' tag"}⏎
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/users \
                                          -H "Content-Type: application/json" \
                                          -d '{
                                        "username": "mohamedlakhdar",
                                        "email": "lakhdar@gmail.com",
                                        "password": "mohamedlakhdar"
                                      }'
{"id":11,"username":"mohamedlakhdar","email":"lakhdar@gmail.com","created_at":"2026-08-22T03:10:12.998763+01:00","updated_at":"2026-08-22T03:10:12.998763+01:00"}⏎
prismo@fedora ~/D/GuildChat (main)>
```
#### prove :
```
 ebug] Listening and serving HTTP on :8080
[GIN] 2026/08/22 - 03:05:13 | 201 | 108.62ms |             ::1 | POST     "/users"
[GIN] 2026/08/22 - 03:05:23 | 400 | 216.03µs |             ::1 | POST     "/users"
[GIN] 2026/08/22 - 03:08:12 | 400 | 194.725µs |             ::1 | POST     "/users"
[GIN] 2026/08/22 - 03:10:13 | 201 | 105.51ms |             ::1 | POST     "/users"
```


#### 1. Understand the layers
for
```
POST /users
```
now we have :
```
Handler
   │
   ├── Request validation
   │     ├── username required
   │     ├── username min/max
   │     ├── email format
   │     └── password min
   │
   ▼
Service
   │
   ├── Hash password
   │
   ▼
Repository
   │
   ▼
PostgreSQL
   │
   └── UNIQUE constraint
```
The repository will encounter a PostgreSQL error when someone tries:
```
username = "lakhdar"
```
and lakhdar already exists.


### done the part the errors of the username and email
```
  postman      go.mod
 internal   Dockerfile   go.sum
prismo@fedora ~/D/G/app (main)> curl -X POST http://localhost:8080/users \

                                      -H "Content-Type: application/json"
\
                                      -d '{
                                    "username": "mohamedlakhdar",
                                    "email": "lakhdar@gmail.com",
                                    "password": "mohamedlakhdar"
                                  }'
{"error":"username already exists"}⏎
prismo@fedora ~/D/G/app (main)> curl -X POST http://localhost:8080/users \

                                      -H "Content-Type: application/json"
\
                                      -d '{
                                    "username": "mohamedlakhda",
                                    "email": "lakhdar@gmail.com",
                                    "password": "mohamedlakhdar"
                                  }'
{"error":"email already exists"}⏎
prismo@fedora ~/D/G/app (main)>
```




At this point the endpoint should handle:


```
POST /users
    │
    ├── JSON parsing
    ├── Request validation
    │     ├── username
    │     ├── email
    │     └── password
    │
    ├── Password hashing (Argon2id)
    │
    ├── PostgreSQL insertion
    │
    ├── Duplicate username → 409
    ├── Duplicate email    → 409
    ├── Unexpected DB error → 500
    │
    └── UserResponse
          └── NO password/hash
```




##### NOW

POST /auth/login

```
POST /auth/login
       │
       ▼
LoginRequest DTO
       │
       ├── email
       └── password
       │
       ▼
Handler
       │
       ▼
Auth Service
       │
       ├── Find user by email
       │
       └── Verify Argon2id password
       │
       ▼
Generate JWT
       │
       ▼
LoginResponse
```


for the jwt we use the asymmetric JWT approach
Private key → sign/issue JWTs
Public key → verify JWTs

For JWT, I'd use RS256 (RSA) or EdDSA (Ed25519). For a learning project, RS256 is very wide used



## Flow:
```
                    AUTH SERVER
                         │
                   Private Key
                         │
                         ▼
POST /auth/login ──→ Generate JWT
                         │
                         │ signed with PRIVATE key
                         ▼
                  ┌─────────────┐
                  │    JWT      │
                  └──────┬──────┘
                         │
                         ▼
                 Client receives it


Later:

Client
  │
  │ Authorization: Bearer <JWT>
  ▼
Protected endpoint
  │
  ▼
JWT Middleware
  │
  │ verify signature
  │ using PUBLIC key
  ▼
    Valid?
   /     \
 yes      no
  │        │
  ▼        ▼
Handler   401
```
