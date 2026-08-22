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
