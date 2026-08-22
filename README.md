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
