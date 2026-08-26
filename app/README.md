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






overall the architecture is like :
```
LOGIN
  │
  ▼
GenerateToken(userID)
  │
  ▼
Claims
  │
  ▼
Sign with PRIVATE KEY
  │
  ▼
JWT
```


```
REQUEST
  │
  ▼
JWT
  │
  ▼
ParseWithClaims()
  │
  ▼
Verify with PUBLIC KEY
  │
  ▼
Claims
  │
  ▼
UserID
```
```
guildchat=# exit
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/users \
                                          -H "Content-Type: application/json" \
                                          -d '{
                                        "username": "mohamedlakhda",
                                        "email": "lakhdar@gmail.com",
                                        "password": "mohamedlakhdar"
                                      }'
{"id":1,"username":"mohamedlakhda","email":"lakhdar@gmail.com","created_at":"2026-08-23T01:56:25.657633Z","updated_at":"2026-08-23T01:56:25.657633Z"}⏎       prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/auth/login \
                                          -H "Content-Type: application/json" \
                                          -d '{
                                        "email": "lakhdar@example.com",
                                        "password": "mohamedlakhdar"
                                      }'
{"error":"invalid credentials"}⏎                                                                                                                             prismo@fedora ~/D/GuildChat (main)> make psql
docker compose exec postgres psql -U postgres -d guildchat
psql (17.11)
Type "help" for help.

guildchat=# select * from users ;
 id |   username    |       email       |                                           password_hash                                           |          created_at           |          updated_at
----+---------------+-------------------+---------------------------------------------------------------------------------------------------+-------------------------------+-------------------------------
  1 | mohamedlakhda | lakhdar@gmail.com | $argon2id$v=19$m=65536,t=1,p=4$Qsqiii+vSPvm6PrTbYlLcA$trZOuUpqsCILnB6dqIOWbwjFg3LOhTqUJVoAtapiXUE | 2026-08-23 01:56:25.657633+00 | 2026-08-23 01:56:25.657633+00
(1 row)

guildchat=# q
guildchat-# /q
guildchat-# \q
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/auth/login \
                                          -H "Content-Type: application/json" \
                                          -d '{
                                        "email": "lakhdar@gmail.com",
                                        "password": "mohamedlakhdar"
                                      }'
{"token":"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc3MiOiJHdWlsZENoYXQiLCJzdWIiOiIxIiwiZXhwIjoxNzg3NTM2Njc1LCJpYXQiOjE3ODc0NTAyNzV9.qDCNkEIKe0svoHCED_SReDKpKNm3A7OAonDI9Ve1P1avjSL-goOg0gb6V6RX7vavVvJv31mOCv2PnbLblEoS7GYbL-xvvR3se8Rd2BxKzABTQpwE7IiMfuFt5JdsPUSfMkMIvroO9Fks4lNNddRqDKuDm88EvnOz2CAhfU0WqgsgZHHHhuP5qyMfAA0OJfyYjV1t_Er01p32I1K8mqX_fDrL4J2Lrs-Afl9GZ82zb6GDU_c-QyOVdxS_rk6Sn6mfqzLfbOqINNTaHXskcGH0oedEz04mSO5m6C0WJZS6VXXb9uWkAfT-P_ItZfbe4HIOA9C4udAK-gef2iEnuZb2SN7JuqYgSvl95vG--K6c016JByYvlvlct6SgC5dGU1yJ7p-KIHPYaPoSFg71Y857Qbbet_YN489cZxn4_wrzCSZ1UgLK6T6jSL6-A4sqGCEjKFKDx_V-Tp6_HfCWWsWUTZV6MxUPuZnGgW1WuySGWIXTM97tEO_xpUaDCpI5DycgRM2K5W_wu7YCDdvXaZxG9vvHizUtgKrRi-eyt98t73YYOsW0d2JUNh_i8pX95pGY1cmHLC-h9IOmzdf4Mzbx_X89S_Zg67qA3W19m7LO1S7TOVUv41JvWE1T-2Jn66VoPlYfZez7r4JzLi8G4M9BELEQ2bUHV4vMqcc974xx3rA","user":{"email":"lakhdar@gmail.com","id":1,"username":"mohamedlakhda"}}⏎

```




delete and put done
```
prismo@fedora ~/D/GuildChat (main)>
prismo@fedora ~/D/GuildChat (main)> curl http://localhost:8080/me \
                                          -H "Authorization: Bearer $TOKEN"
{"id":1,"username":"mohamedlakhda","email":"lakhdar@gmail.com","created_at":"2026-08-23T01:56:25.657633Z","updated_at":"2026-08-23T01:56:25.657633Z"}⏎
prismo@fedora ~/D/GuildChat (main)> curl -X PUT  http://localhost:8080/me \
                                          -H "Authorization: Bearer $TOKEN" -d '{"username":"ana mohamed"}'
{"id":1,"username":"ana mohamed","email":"","created_at":"2026-08-23T01:56:25.657633Z","updated_at":"2026-08-23T04:07:42.956584Z"}⏎
prismo@fedora ~/D/GuildChat (main)> curl -x DELETE http://localhost:8080/me
curl: (5) Could not resolve proxy: DELETE
prismo@fedora ~/D/GuildChat (main)> curl -X DELETE http://localhost:8080/me
{"error":"Authorization header is missing"}⏎
prismo@fedora ~/D/GuildChat (main)> curl -X DELETE http://localhost:8080/me -H "Authorization: Bearer $TOKEN"
prismo@fedora ~/D/GuildChat (main)> curl -X PUT  http://localhost:8080/me \
                                          -H "Authorization: Bearer $TOKEN" -d '{"username":"ana mohamed"}'
{"error":"user not found"}⏎
prismo@fedora ~/D/GuildChat (main)>
```





for the friendship :
```
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/auth/login \

                                          -H "Content-Type: application/json" \
                                          -d '{
                                        "email": "user1@gmail.com",
                                        "password": "secret123"
                                      }'
{"token":"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc3MiOiJHdWlsZENoYXQiLCJzdWIiOiIxIiwiZXhwIjoxNzg3ODI1NDQ0LCJpYXQiOjE3ODc3MzkwNDR9.FNpHF3xTMbECOVXOtk-gqM3PtH8vnC4FtWODxTinhIMojp01cYcLtnIAvf9iDXOX5smSPe3qjrjanal3iQ2TOFrPMNe9xHHK2HRRpIG5pB8ZSUdpm9-9FEx0FSR2q2Fnxlcna6ahd7zEA6Ci92U71bIzblD_GvivXVxBpRchywzwwkfkaN3S-HfhWyrkmhQJRNSS92-kBvxZzg_niKQNyQS7BJtja-q7kc1qwsOqv3ArI9RcMHsuXqnBQv8WbeC6Q-eYKXZihC7o5wtEuG7EEL0lq3ILhyEK7rLQ2RUyho0nAk8CCBVQWR1LUvS2KCDxiApMKqVnCSrEyUIoiAO_FovB1AqfnXIliMsYbZV51CGXwA7SgB5EHWBE1Qzj_k5JfZJ7pQGYRfV-URsvV-JsN7OKIQD6CtQNFCNU9l-Pnb085AFM86T0y5oUWOj6kk9ly64FdfMRhT4jBYTVZw5XfIXg16bHHtFo6O8Mx3iHKbhRyJIofWU-7RelU0vCzKGAoTz0rNvlW3Sm4J6VNNIKgrCCCl8Jgg19Q0Vh1urxsbGfQ-dfpVNAl_e1C56mLT5yDV77Ed4PJU6XlC0cF7LylKYvHUD8_mYL7GGKb8X_RlYJAWytv-4kfe9RTe6XooQxpFoXiABy2MUkiRbVqPIBo0gvTwKVRv0GtrV8Zprismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/users/3/friend-request -H "Authorization: Bearer "
{"error":"Authorization header format must be Bearer {token}"}⏎
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/auth/login \
                                          -H "Content-Type: application/json" \
                                          -d '{
                                        "email": "user1@gmail.com",
                                        "password": "secret123"
                                      }'
{"token":"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc3MiOiJHdWlsZENoYXQiLCJzdWIiOiIxIiwiZXhwIjoxNzg3ODI1NTEzLCJpYXQiOjE3ODc3MzkxMTN9.rAXWYVPkb7xcgV43yuvYNFbJ_PMp7R20Kzcr8A78-4kBhg4xR3LkZwkRuQ5ddqSXNBZYDnLf-EmbIAy9eWJKdmBPLSgtDlFyoU8KQOWDyCQSGeowCjqtliCWoGuGnsWtLEE3vz53MpxOc07Io8BLPbY91mC9g_2yO6PSspaLmCOWxzuZUGhZP5NO7WW-YwhPdXMTVTucx4IN27cdrUx0jmw2hmHED-cCU5f0ywOH-HSwYyn0HE4tMBNMT8nwthF9MU1wpAzuyc3y9s1RS8vQ-dgDka6-9e7ZewdmSYgushk048eyns9h2iknn9-59DkWWNURYfjQNfBx2RlBSFPnPhXUgfAial0wHYEVkwrmXXtIcdW7j7hbCcOA6FyFbo-WVb-pt3pRyIlff_oDJRnK18z8m1vCzwaNYZ8v-t3wrf-V_G9oVGO5nu1l9Hbjrj9Gtu44OoffASU7ijjiLqVX1OFl30xhSF53EvqQMu7bD0U-uUYXQF2bDTebzX4V61ywl2pXOEdi-OefqIMLJ470lXGBKMZVg7qQeR1mrPkVp3Q6CsMNFYi7PhKs1FVwVcuLaC2NbKJGt51nIEYwp4ZfKmNhdo5PVQgE1nSbUZCQl4xlR6wKL_YW0CTarfMiUDisEwEBVoB4uzY1meGCKy0Vh8UAq5aSOgBG9ioY65aQGOo","user":{"email":"user1@gmail.com","id":1,"username":"user1"}}⏎
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/users/3/friend-request -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc3MiOiJHdWlsZENoYXQiLCJzdWIiOiIxIiwiZXhwIjoxNzg3ODI1NTEzLCJpYXQiOjE3ODc3MzkxMTN9.rAXWYVPkb7xcgV43yuvYNFbJ_PMp7R20Kzcr8A78-4kBhg4xR3LkZwkRuQ5ddqSXNBZYDnLf-EmbIAy9eWJKdmBPLSgtDlFyoU8KQOWDyCQSGeowCjqtliCWoGuGnsWtLEE3vz53MpxOc07Io8BLPbY91mC9g_2yO6PSspaLmCOWxzuZUGhZP5NO7WW-YwhPdXMTVTucx4IN27cdrUx0jmw2hmHED-cCU5f0ywOH-HSwYyn0HE4tMBNMT8nwthF9MU1wpAzuyc3y9s1RS8vQ-dgDka6-9e7ZewdmSYgushk048eyns9h2iknn9-59DkWWNURYfjQNfBx2RlBSFPnPhXUgfAial0wHYEVkwrmXXtIcdW7j7hbCcOA6FyFbo-WVb-pt3pRyIlff_oDJRnK18z8m1vCzwaNYZ8v-t3wrf-V_G9oVGO5nu1l9Hbjrj9Gtu44OoffASU7ijjiLqVX1OFl30xhSF53EvqQMu7bD0U-uUYXQF2bDTebzX4V61ywl2pXOEdi-OefqIMLJ470lXGBKMZVg7qQeR1mrPkVp3Q6CsMNFYi7PhKs1FVwVcuLaC2NbKJGt51nIEYwp4ZfKmNhdo5PVQgE1nSbUZCQl4xlR6wKL_YW0CTarfMiUDisEwEBVoB4uzY1meGCKy0Vh8UAq5aSOgBG9ioY65aQGOo"
{"id":1,"requester_id":1,"receiver_id":3,"status":"pending","created_at":"2026-08-26T10:12:07.184571Z","updated_at":"2026-08-26T10:12:07.184571Z"}⏎
prismo@fedora ~/D/GuildChat (main)> curl -X POST http://localhost:8080/users/3/friend-request -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJpc3MiOiJHdWlsZENoYXQiLCJzdWIiOiIxIiwiZXhwIjoxNzg3ODI1NTEzLCJpYXQiOjE3ODc3MzkxMTN9.rAXWYVPkb7xcgV43yuvYNFbJ_PMp7R20Kzcr8A78-4kBhg4xR3LkZwkRuQ5ddqSXNBZYDnLf-EmbIAy9eWJKdmBPLSgtDlFyoU8KQOWDyCQSGeowCjqtliCWoGuGnsWtLEE3vz53MpxOc07Io8BLPbY91mC9g_2yO6PSspaLmCOWxzuZUGhZP5NO7WW-YwhPdXMTVTucx4IN27cdrUx0jmw2hmHED-cCU5f0ywOH-HSwYyn0HE4tMBNMT8nwthF9MU1wpAzuyc3y9s1RS8vQ-dgDka6-9e7ZewdmSYgushk048eyns9h2iknn9-59DkWWNURYfjQNfBx2RlBSFPnPhXUgfAial0wHYEVkwrmXXtIcdW7j7hbCcOA6FyFbo-WVb-pt3pRyIlff_oDJRnK18z8m1vCzwaNYZ8v-t3wrf-V_G9oVGO5nu1l9Hbjrj9Gtu44OoffASU7ijjiLqVX1OFl30xhSF53EvqQMu7bD0U-uUYXQF2bDTebzX4V61ywl2pXOEdi-OefqIMLJ470lXGBKMZVg7qQeR1mrPkVp3Q6CsMNFYi7PhKs1FVwVcuLaC2NbKJGt51nIEYwp4ZfKmNhdo5PVQgE1nSbUZCQl4xlR6wKL_YW0CTarfMiUDisEwEBVoB4uzY1meGCKy0Vh8UAq5aSOgBG9ioY65aQGOo"
{"error":"friendship already exists"}⏎

```
