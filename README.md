```
                    ┌──────────────────┐
                    │    Frontend      │
                    │   React/Vite     │
                    │    (later)       │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │      Nginx       │
                    │  reverse proxy   │
                    │     (later)      │
                    └────────┬─────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │   Go Backend     │
                    │     Gin API      │
                    │    :8080         │
                    └───────┬──────────┘
                            │
                    ┌───────┴──────────┐
                    ▼                  ▼
             ┌──────────────┐   ┌──────────────┐
             │  PostgreSQL  │   │    MongoDB   │
             │    :5432     │   │    :27017    │
             └──────────────┘   └──────────────┘
```



adding middleware to get thing done like this :
```
GET /users/:id          public
POST /users             public
POST /auth/login        public

GET /me                 🔒 JWT required
PUT /users/:id          🔒 JWT required
DELETE /users/:id       🔒 JWT required
```
***
At this point the JWT core is done:

RSA private/public keys loaded
JWTManager implemented
GenerateToken() signs with the private key
ValidateToken() verifies with the public key
Claims contain user_id, iss, sub, iat, exp
JWT middleware extracts Bearer <token>
Middleware stores user_id in Gin's context
Login issues the JWT
Passwords use Argon2id
Registration/login flow works


***




testing the GET /me
```
prismo@fedora ~/D/GuildChat (main)> set TOKEN (curl -s -X POST http://localhost:8080/auth/login \
                                              -H "Content-Type: application/json" \
                                              -d '{
                                        "email": "lakhdar@gmail.com",
                                        "password": "mohamedlakhdar"
                                      }' | jq -r '.token')
prismo@fedora ~/D/GuildChat (main)> echo $TOKEN

eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc3MiOiJHdWlsZENoYXQiLCJzdWIiOiIxIiwiZXhwIjoxNzg3NTQxMjY4LCJpYXQiOjE3ODc0NTQ4Njh9.w9xdo-zhcMQLP_d5-Lbv8gQhm6YYxWXGYr3hRQ0Zv4bnlUZ0RsGCLxKWv05ke7HOdDfIag82eaBCZiGOwSPRH00OUkUL-vl5fOprckQfwulnUrqtNXZYB-E_UGk_aNdmKgxcB0zt3VibSpRFvKe52OxPKZEtQmjHve4TU-fp_iL7Ozl4PpsRiQrJiAC6251xsTOscY3OR-QZ0DW5JtcwGqiK44PP0yjo0yB8KJMOEYPAn8zPu-6oS31fPLhDgmyyNNpLeGOigrkMIrCpLUeU7YRbEw7GvmfzKZMc4KrdYMOn5L74P6yYRJ2nOt4Cig850R-57rfP15YV8qxg5U0XKjojYj7NxU1Rre3hvvQ5Dh1IE9Bm6DoubEA1uoCf12iCaOTIMopai-m-HUrmVbD4_05a97bZtsJYJoX-qMPNbAggY1JMZrR2LhKq4PRVwT3F7UwAfHU4SXvgMIZmkhCBepu3a3s6kaq3yhOn-kwPBorkXAK6ZLOMIqqFBeemEiMOQJ_N9C4SfsPDWBmkx5zoBeIxoZ4JxNVCTJbQV-5A9Bz1fa68nB-zTB_qDzoRJMrivFw6Aie1QwY2LryPup-8YYBOWeYumpparX-3uWUetIbFtz8ztFw22dY2VssR-KyUJx-LuY9JrXjp-RhxYk6YxebY2mRSA_kJxeUwyhSnkd8
prismo@fedora ~/D/GuildChat (main)> curl http://localhost:8080/me \
                                          -H "Authorization: Bearer $TOKEN"
{"id":1,"username":"mohamedlakhda","email":"lakhdar@gmail.com","created_at":"2026-08-23T01:56:25.657633Z","updated_at":"2026-08-23T01:56:25.657633Z"}⏎
prismo@fedora ~/D/GuildChat (main)> curl http://localhost:8080/me
{"error":"Authorization header is missing"}⏎
prismo@fedora ~/D/GuildChat (main)> curl http://localhost:8080/me \
                                          -H "Authorization: Bearer bullshit"
{"error":"Invalid token"}⏎
prismo@fedora ~/D/GuildChat (main)> curl http://localhost:8080/me \
                                          -H "Authorization: Bearer $TOKEN"
{"id":1,"username":"mohamedlakhda","email":"lakhdar@gmail.com","created_at":"2026-08-23T01:56:25.657633Z","updated_at":"2026-08-23T01:56:25.657633Z"}⏎
prismo@fedora ~/D/GuildChat (main)>
```



clean architecture :
```
                    ┌───────────────┐
                    │ POST /login   │
                    └───────┬───────┘
                            │
                     Verify password
                            │
                            ▼
                    ┌───────────────┐
                    │ Generate JWT  │
                    │  Private Key │
                    └───────┬───────┘
                            │
                            ▼
                         Client
                            │
                 Authorization: Bearer
                            │
                            ▼
                    ┌───────────────┐
                    │ JWT Middleware│
                    └───────┬───────┘
                            │
                     Public Key
                            │
                     Verify signature
                            │
                            ▼
                       user_id = 1
                            │
                            ▼
                       GET /me
                            │
                            ▼
                     UserRepository
                            │
                            ▼
                       PostgreSQL

```

***

Schema changes go through migrations — see [docs/MIGRATIONS.md](docs/MIGRATIONS.md)
for the concepts, a walkthrough of the code, and the commands.
