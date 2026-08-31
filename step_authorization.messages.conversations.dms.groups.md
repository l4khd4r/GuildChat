# Project Notes — Messaging Backend Architecture

## Core Decision

Keep WebSockets for later. Build rooms/DMs + roles/permissions first.

**Key point:** WebSockets are a *transport mechanism*. Rooms, DMs, participants, roles, and permissions are *domain logic*. Settle the domain logic before making it real-time.

---

## 1. Recommended Order

| # | Step | Status |
|---|------|--------|
| 1 | Users + Auth | ✅ |
| 2 | Friendships | ✅ |
| 3 | Conversations / DMs | ← next |
| 4 | Rooms / Channels | |
| 5 | Roles + Permissions | |
| 6 | Messages REST API | |
| 7 | WebSocket layer | ← then |
| 8 | Real-time events | |
| 9 | Presence / typing / receipts | |

---

## 2. Why Not WebSockets Now?

If you implement WebSockets immediately, you'll end up asking:

> "When a WebSocket message arrives, what is this user allowed to do?"

Then you'll have to build the authorization, room membership, conversation model, etc. **while simultaneously debugging WebSocket connections.** That's unnecessary complexity.

### Make this work first

```
POST /conversations/:id/messages
```

with:

```
JWT
 ↓
userID from context
 ↓
verify participant
 ↓
create message
 ↓
return message
```

### Once that is solid, WebSocket becomes straightforward

```
WebSocket message
       ↓
authenticate user
       ↓
identify conversation/room
       ↓
check permission
       ↓
save message
       ↓
broadcast event
```

---

## 3. One Change to the Plan

**Don't build DMs and rooms as completely separate systems.** Model them around a common concept:

```
                 Conversation
                /            \
              DM             Room
              │                │
        2 participants     N participants
                              │
                           members
                              │
                           roles
```

---

## 4. Schema

### conversations

```
conversations
----------------
id
type            -- dm / room
name            -- nullable for DM
created_by
created_at
updated_at
```

### conversation_members

```
conversation_members
--------------------
conversation_id
user_id
role
joined_at
```

### messages

```
messages
----------------
id
conversation_id
sender_id
content
created_at
updated_at
```

➡️ Now DMs and rooms use the same message infrastructure.

---

## 5. Roles Come Naturally Afterward

For a room:

```
owner
admin
member
```

Then later, permissions:

```
SEND_MESSAGE
DELETE_MESSAGE
KICK_MEMBER
BAN_MEMBER
MANAGE_ROOM
...
```

You don't need an elaborate RBAC system immediately. **Start simple.**

---

## 6. Final Verdict

Don't do WebSockets yet. Do:

```
        ┌──────────────┐
        │ Conversations│
        └──────┬───────┘
               │
       ┌───────┴────────┐
       │                │
      DMs             Rooms
       │                │
       └───────┬────────┘
               │
          Participants
               │
             Roles
               │
           Messages
               │
               ▼
          WebSockets
```

That gives you a much cleaner backend architecture. When you finally implement WebSockets, you're **adding real-time delivery to an already-working messaging system**, rather than building the entire system inside the WebSocket layer.
