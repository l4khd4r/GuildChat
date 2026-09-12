# WebSocket Phase 1 — implementation plan and complete code

Copy/paste reference for GuildChat. Nothing in here has been written into the
repository; every file below is yours to place.

State of the tree when this was written:

- `app/internal/ws/event.go` exists but contains only its package line.
- `gorilla/websocket` is not in `go.mod` yet.

---

## Deviations from the spec, and why

Three things differ from the brief, each because one of its own rules forced it.

1. **The envelope example showed `content` and `sender_id`.** `dto.MessageResponse`
   has `body` and a nested `sender` object. The rule "do not create a separate
   message representation for WebSockets" wins, so the real frame carries the
   DTO's field names.

2. **`map[userID]*Client` means one tab per user.** Implemented exactly as
   specified. Opening a second tab evicts the first, which is immediately
   visible when testing with two windows. The one-line change to allow several
   is noted in `hub.go`.

3. **No repository change.** Membership comes from the existing
   `ConversationRepository.ListMembers` through a one-method interface, so the
   hub is testable without Postgres and `conversation_repository.go` is left
   alone.

---

## File map

```text
CREATE:
app/internal/ws/hub.go                       the registry of connected users, and the fan-out
app/internal/ws/client.go                    one connection, its single writer goroutine
app/internal/ws/handler.go                   GET /ws: authenticate, upgrade, register
app/internal/ws/hub_test.go                  registration, eviction, fan-out targeting
app/internal/ws/handler_test.go              real sockets against a real JWT manager
app/internal/handler/message_ws_db_test.go   full stack, DB-guarded
docs/WEBSOCKET-PHASE-1.md                    the final document

MODIFY:
app/internal/ws/event.go                     currently one line; becomes the envelope
app/internal/handler/message_handler.go      holds the hub, publishes after the 201
app/internal/router/router.go                one route, outside the protected group
app/cmd/server/main.go                       build the hub, wire it into both
app/go.mod, app/go.sum                       gorilla/websocket
```

First, the dependency:

```bash
cd app && go get github.com/gorilla/websocket@v1.5.3
```

Commit both `go.mod` and `go.sum`. CI runs `go mod tidy` followed by
`git diff --exit-code` on them.

---

## FILE: app/internal/ws/event.go
## ACTION: MODIFY

```go
package ws

import "github.com/l4khd4r/GuildChat/internal/dto"

// Envelope is the shape of every frame the server pushes. One outer type,
// discriminated by Type, so a client parses the wrapper once and switches on
// what is inside.
//
// Adding typing, read_receipt or presence_changed later is a new constant and
// a new Data struct. It is not a new frame format, and a client written today
// can ignore an event type it has never heard of instead of failing to parse.
type Envelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// EventMessageCreated is the only event Phase 1 emits. It fires after a
// message has been committed by POST /conversations/:id/messages.
const EventMessageCreated = "message_created"

// MessageCreatedData is the payload of a message_created frame.
//
// The message is a dto.MessageResponse: the exact type the send and list
// endpoints already return. There is deliberately no socket-specific message
// shape, so a client renders a message with one function regardless of how it
// arrived, and a field added to the HTTP contract cannot silently diverge from
// the pushed one.
//
// Note what that means on the wire: the fields are the DTO's, so the body is
// "body" and the sender is a nested user object, not a "sender_id".
//
// It is a struct wrapping the message rather than the bare message because
// "data" is an object with room to grow. An event that later needs to say why
// a message appeared adds a field here without breaking frames already shipped.
type MessageCreatedData struct {
	Message dto.MessageResponse `json:"message"`
}

// NewMessageCreatedEvent wraps a message response in its envelope.
func NewMessageCreatedEvent(message dto.MessageResponse) Envelope {
	return Envelope{
		Type: EventMessageCreated,
		Data: MessageCreatedData{Message: message},
	}
}
```

A frame on the wire:

```json
{
  "type": "message_created",
  "data": {
    "message": {
      "id": 123,
      "conversation_id": 42,
      "sender": { "id": 7, "username": "rogue", "email": "r@example.com",
                  "created_at": "...", "updated_at": "..." },
      "body": "hello",
      "client_msg_id": "0d1c…",
      "created_at": "2026-09-12T04:51:00Z"
    }
  }
}
```

```text
Function: NewMessageCreatedEvent
Purpose: Build the one event Phase 1 emits.
Parameters: the message response already built for the HTTP 201.
Returns: an Envelope ready to marshal.
Called by: MessageHandler.broadcastMessageCreated.
Why it exists: so the event type string is written once, not at every call site.
Concurrency: none, pure value construction.
Failure behavior: cannot fail.
```

---

## FILE: app/internal/ws/client.go
## ACTION: CREATE

```go
package ws

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait is how long one write may block before the connection is
	// considered dead. Short on purpose: a write that cannot finish in ten
	// seconds is not going to finish.
	writeWait = 10 * time.Second

	// pongWait is how long we tolerate silence from a client. In Phase 1 the
	// client never sends anything, so the ping/pong exchange below is the only
	// evidence the connection is alive. Without it, a laptop that went into a
	// tunnel holds a goroutine and a send buffer until the process restarts.
	pongWait = 60 * time.Second

	// pingPeriod must be meaningfully shorter than pongWait, or we time out
	// waiting for a pong we have not asked for yet.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize caps an inbound frame. Phase 1 reads nothing useful, so
	// this is purely a guard: a hostile client must not be able to make the
	// server allocate on its behalf.
	maxMessageSize = 1024

	// sendBuffer is how many frames may queue for one connection before it is
	// declared too slow and dropped. A chat client that cannot drain a few
	// dozen messages is not going to catch up, and the alternative to dropping
	// it is letting it slow down the broadcast to everyone else.
	sendBuffer = 64
)

// Client is one authenticated WebSocket connection, owned by one user.
//
// Writes never touch conn directly. They go into send, and writePump is the
// only goroutine that writes to the socket. That is the whole concurrency
// design: gorilla permits exactly one concurrent writer per connection, and a
// single owning goroutine is a cheaper way to guarantee that than a mutex held
// across a network write.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID int64

	// send carries already-marshalled frames from any goroutine to writePump.
	send chan []byte

	// closeOnce guards send against a double close. Both pumps can decide
	// independently that this connection is finished, and Unregister may be
	// called more than once for the same client.
	closeOnce sync.Once
}

// newClient builds a client for an already-upgraded connection.
func newClient(hub *Hub, conn *websocket.Conn, userID int64) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, sendBuffer),
	}
}

// UserID reports which user this connection belongs to.
func (c *Client) UserID() int64 {
	return c.userID
}

// enqueue hands one marshalled frame to this connection's writer.
//
// It never blocks. A full buffer means the client is not draining, and the
// caller is a broadcast loop holding the hub's read lock, so blocking here
// would stall delivery to every other member of the conversation. A false
// return means "drop this client", and the caller does exactly that.
func (c *Client) enqueue(payload []byte) bool {
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

// close shuts the connection down by closing its send channel, which is the
// signal writePump waits for. Safe to call any number of times from any
// goroutine.
func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

// writePump owns the write side of the connection for its whole life.
//
// It drains send, and it sends a periodic ping so that a connection which has
// gone away without a close frame is noticed. It exits when send is closed or
// any write fails, and closing conn on the way out is what unblocks readPump.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case payload, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// close() ran: say goodbye politely and stop.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump owns the read side.
//
// Phase 1 is server to client only, so everything a client sends is discarded.
// The loop exists anyway because it is what notices a hung-up browser: without
// a reader, a dead connection would sit in the hub until the next failed write.
// Its deferred Unregister is the single place a connection leaves the hub.
//
// An unrecognised inbound frame is ignored rather than treated as an error. A
// client that learns to send something in Phase 2 must not be disconnected by
// a server that has not been deployed yet.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		// Every pong buys another pongWait of silence.
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws: user %d read error: %v", c.userID, err)
			}
			return
		}
	}
}
```

```text
Function: enqueue
Purpose: Queue one frame for this connection without blocking.
Parameters: a marshalled frame.
Returns: false when the buffer is full, meaning the client should be dropped.
Called by: Hub.BroadcastToConversation, under the hub's read lock.
Why it exists: one slow reader must not stall the fan-out to everyone else.
Concurrency: a channel send, safe from any goroutine.
Failure behavior: returns false; never blocks, never panics.

Function: close
Purpose: Signal writePump to finish and tear the connection down.
Parameters: none.
Returns: nothing.
Called by: Hub.Unregister, and Hub.Register when evicting a previous socket.
Why it exists: closing a channel twice panics, and two paths can both close.
Concurrency: sync.Once, safe and idempotent.
Failure behavior: cannot fail.

Function: writePump
Purpose: Be the only goroutine that writes to this socket.
Parameters: none.
Returns: nothing; runs until the connection ends.
Called by: Handler.ServeWS, in its own goroutine.
Why it exists: gorilla allows exactly one concurrent writer per connection.
Concurrency: sole owner of the write side; also sends keepalive pings.
Failure behavior: any write error ends it and closes conn, which ends readPump.

Function: readPump
Purpose: Detect a dead or closed connection and unregister it.
Parameters: none.
Returns: nothing; runs until the connection ends.
Called by: Handler.ServeWS, on the request goroutine.
Why it exists: without a reader, a hung-up client stays in the hub.
Concurrency: sole owner of the read side.
Failure behavior: any read error exits the loop and unregisters the client.
```

---

## FILE: app/internal/ws/hub.go
## ACTION: CREATE

```go
package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/l4khd4r/GuildChat/internal/model"
)

// MemberLister is the only thing the hub needs from the database: who is in a
// conversation. *repository.ConversationRepository already satisfies it.
//
// It is an interface rather than the concrete repository for two reasons. The
// ws package stays independent of the repository package, and the fan-out can
// be tested against a fixed roster with no Postgres running.
type MemberLister interface {
	ListMembers(ctx context.Context, conversationID int64) ([]*model.ConversationMemberEntry, error)
}

// Hub is the registry of connected users.
//
// One entry per user, as specified. A user with no open tab has no entry at
// all, so the map is the size of the online population rather than the users
// table. Registering a second connection for the same user evicts the first;
// to allow several tabs instead, change the value type to a
// map[*Client]struct{} and adjust Register, Unregister and the loop in
// BroadcastToConversation.
//
// There is deliberately no conversation-to-clients index. Membership already
// lives in Postgres, and a second copy in memory would have to be updated by
// every add, remove and future ban, going silently stale the first time one of
// them forgets. Broadcasting costs one indexed read instead.
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]*Client

	members MemberLister
}

// NewHub builds an empty hub that reads conversation rosters from members.
func NewHub(members MemberLister) *Hub {
	return &Hub{
		clients: make(map[int64]*Client),
		members: members,
	}
}

// Register adds an authenticated client to the hub.
//
// If that user already had a connection, the old one is closed and replaced.
// The close happens after the lock is released, because closing touches the
// evicted client rather than the map and there is no reason to hold every
// other broadcast up while it happens.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	previous, existed := h.clients[client.userID]
	h.clients[client.userID] = client
	h.mu.Unlock()

	if existed && previous != client {
		log.Printf("ws: user %d reconnected, closing the previous connection", client.userID)
		previous.close()
	}
}

// Unregister removes a client from the hub and closes it.
//
// The identity check matters. A client whose readPump exits late must not
// delete the entry belonging to the reconnection that already replaced it, or
// a user who reconnects quickly ends up registered as nobody.
//
// Safe to call more than once for the same client.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if current, ok := h.clients[client.userID]; ok && current == client {
		delete(h.clients, client.userID)
	}
	h.mu.Unlock()

	client.close()
}

// ClientFor returns the connection belonging to a user, if they have one.
func (h *Hub) ClientFor(userID int64) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[userID]
	return client, ok
}

// Online reports how many users currently have a connection.
func (h *Hub) Online() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

// BroadcastToConversation sends one event to every member of a conversation
// who is currently connected.
//
// The roster comes from the database, which is the source of truth for
// membership: someone added a second ago receives this, and someone removed a
// second ago does not, with no in-memory state to keep in step.
//
// Members who are offline receive nothing and are not an error. They will see
// the message when they next call GET /conversations/:id/messages.
//
// It returns nothing, on purpose. By the time this runs the message is
// committed, so no socket failure may turn a successful request into a failed
// one. Everything that goes wrong here is logged and dropped.
func (h *Hub) BroadcastToConversation(ctx context.Context, conversationID int64, event Envelope) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("ws: marshal %s: %v", event.Type, err)
		return
	}

	members, err := h.members.ListMembers(ctx, conversationID)
	if err != nil {
		log.Printf("ws: roster for conversation %d: %v", conversationID, err)
		return
	}

	// Slow clients are collected here and dropped after the lock is released.
	// Unregister takes the write lock, so doing it inline would deadlock.
	var slow []*Client

	h.mu.RLock()
	for _, member := range members {
		client, ok := h.clients[member.User.ID]
		if !ok {
			continue // offline: nothing to do, and not a failure
		}
		if !client.enqueue(payload) {
			slow = append(slow, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range slow {
		log.Printf("ws: dropping slow connection for user %d", client.userID)
		h.Unregister(client)
	}
}
```

```text
Function: Register
Purpose: Put an authenticated connection in the map, replacing any previous one.
Parameters: the client built after a successful upgrade.
Returns: nothing.
Called by: Handler.ServeWS.
Why it exists: the hub is the only thing that knows who is reachable.
Concurrency: write lock held only around the map; the eviction close is outside it.
Failure behavior: cannot fail.

Function: Unregister
Purpose: Remove a connection and close it.
Parameters: the client leaving.
Returns: nothing.
Called by: Client.readPump on exit, and BroadcastToConversation for slow clients.
Why it exists: single exit point, so a dead socket cannot linger in the map.
Concurrency: write lock; identity-checked so a late exit cannot evict a reconnect.
Failure behavior: idempotent, safe on a client already gone.

Function: ClientFor
Purpose: Find a user's connection.
Parameters: a user id.
Returns: the client and whether one exists.
Called by: tests, and anything in Phase 2 needing a single recipient.
Why it exists: section 6 of the spec asks for lookup by user id.
Concurrency: read lock.
Failure behavior: returns false when the user is offline.

Function: BroadcastToConversation
Purpose: Push one event to the connected members of a conversation.
Parameters: a context for the roster read, the conversation id, the event.
Returns: nothing.
Called by: MessageHandler.broadcastMessageCreated.
Why it exists: the whole point of the hub.
Concurrency: read lock over the fan-out; slow clients dropped after release.
Failure behavior: marshal and roster errors are logged and swallowed; delivery is best effort.
```

---

## FILE: app/internal/ws/handler.go
## ACTION: CREATE

```go
package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/l4khd4r/GuildChat/internal/auth"
)

// Handler upgrades GET /ws into a socket.
//
// It cannot sit behind jwtManager.Middleware(). The browser WebSocket API
// cannot set an Authorization header on a handshake, so the token arrives as a
// query parameter and is validated here with the same JWTManager the rest of
// the API uses.
//
// TODO: production should not authenticate a socket with the JWT in the URL.
// Query strings land in access logs, proxy logs and browser history, so the
// token leaks to anywhere logs are shipped. The replacement is a short-lived
// single-use ticket: POST /ws/ticket behind the normal middleware returns an
// opaque id valid for a few seconds and usable once, and this handler
// exchanges it for a user id instead of parsing a JWT. Nothing below changes
// shape when that lands. Not implemented in Phase 1.
type Handler struct {
	hub        *Hub
	jwtManager *auth.JWTManager
	upgrader   websocket.Upgrader
}

// NewHandler builds the /ws handler.
func NewHandler(hub *Hub, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		hub:        hub,
		jwtManager: jwtManager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,

			// Any origin is accepted, and that is safe specifically because
			// this handshake is authenticated by a token in the URL rather
			// than by a cookie. A hostile page can open a socket to us, but it
			// has no way to obtain the victim's token to put in it, so it only
			// ever authenticates as itself.
			//
			// TODO: the moment the ticket above becomes a cookie-backed
			// session, this must become a real allow-list, or that page can
			// open an authenticated socket as the victim.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// ServeWS authenticates the caller, upgrades the connection, and registers it.
//
// Everything before Upgrade answers with ordinary JSON, because no upgrade has
// happened yet. Nothing after it can: once Upgrade returns, the response has
// been hijacked and writing to the gin context would panic.
func (h *Handler) ServeWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token is required"})
		return
	}

	claims, err := h.jwtManager.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade has already written its own error response.
		log.Printf("ws: upgrade for user %d: %v", claims.UserID, err)
		return
	}

	client := newClient(h.hub, conn, claims.UserID)
	h.hub.Register(client)
	log.Printf("ws: user %d connected (%d online)", client.userID, h.hub.Online())

	go client.writePump()

	// readPump runs on this goroutine rather than a new one. The request
	// goroutine is already here, and blocking it is what holds the connection
	// open for the life of the socket.
	client.readPump()
	log.Printf("ws: user %d disconnected (%d online)", client.userID, h.hub.Online())
}
```

```text
Function: ServeWS
Purpose: Turn an authenticated GET /ws into a registered connection.
Parameters: the gin context; the token comes from ?token=.
Returns: nothing; blocks until the socket closes.
Called by: the router, outside the protected group.
Why it exists: the browser cannot send an Authorization header on a handshake.
Concurrency: starts writePump, then becomes readPump for this connection.
Failure behavior: 401 on a missing or invalid token; a failed upgrade is logged and dropped.
```

---

## FILE: app/internal/handler/message_handler.go
## ACTION: MODIFY

Complete file. The only changes are the struct field, the constructor, four
lines at the end of `SendMessage`, and the new method. `ListMessages` is
untouched.

```go
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/dto"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/service"
	"github.com/l4khd4r/GuildChat/internal/ws"
)

// broadcastTimeout bounds the roster read the fan-out needs. It runs after the
// response has been written, on a context of its own, so it needs a deadline
// of its own or a stalled database would leak a goroutine per message.
const broadcastTimeout = 5 * time.Second

type MessageHandler struct {
	messageService *service.MessageService
	hub            *ws.Hub
}

func NewMessageHandler(messageService *service.MessageService, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{messageService: messageService, hub: hub}
}

// SendMessage posts a message to a conversation the caller is a member of.
//
// 404 covers both a conversation that does not exist and one the caller is not
// in, so the endpoint cannot be used to probe ids. There is no 403 here at all:
// membership is the only permission, and a member who is refused would have
// nothing to be refused for.
//
// Sending the same client_msg_id twice returns the original message rather than
// creating a second one, so a client retrying after a dropped connection is
// safe to do so.
//
// Once the message is committed it is also pushed to every connected member
// over the socket. That push cannot affect the response: it happens after the
// 201 is written, and it reports no errors upward.

func (h *MessageHandler) SendMessage(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	request := dto.SendMessageRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	message, err := h.messageService.SendMessage(c.Request.Context(), conversationID, userID, request.Body, request.ClientMsgID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		case errors.Is(err, repository.ErrMessageBodyRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "message body is required"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	// One MessageResponse, used for both destinations. The socket cannot drift
	// from the HTTP contract because it is literally the same value.
	response := toMessageResponse(message)
	c.JSON(http.StatusCreated, gin.H{"message": response})

	h.broadcastMessageCreated(conversationID, response)
}

// broadcastMessageCreated pushes a committed message to the conversation's
// connected members.
//
// It runs after the 201 has been written, so the roster read never delays the
// sender's response.
//
// The context is a fresh one rather than c.Request.Context(). The request
// context is cancelled the moment the sender's connection goes away, and a
// sender who posts and immediately closes the tab would otherwise cancel
// delivery to everyone else in the room, for a message that is already stored.
//
// Nothing is returned and nothing can fail here in a way the caller sees. The
// message is in Postgres; anyone who misses the frame catches up with
// GET /conversations/:id/messages.
func (h *MessageHandler) broadcastMessageCreated(conversationID int64, message dto.MessageResponse) {
	if h.hub == nil {
		return // no socket wired in, for tests that only exercise HTTP
	}

	ctx, cancel := context.WithTimeout(context.Background(), broadcastTimeout)
	defer cancel()

	h.hub.BroadcastToConversation(ctx, conversationID, ws.NewMessageCreatedEvent(message))
}

/*
	ListMessages returns once page of a conversation the caller is a member of,
	newest first.

	Same 404 rule as SendMEssage: a conversation that does not exist and one the
	caller is not in are answered identically

	A malformed `before` or `limit` is a 400 rather than a slient fallback to the
	defaults. A client that sent a cursor and got the newest page back instead
	would loop forever without ever seeing an error.
*/

func (h *MessageHandler) ListMessages(c *gin.Context) {
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation id"})
		return
	}

	query := dto.ListMessagesQuery{}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
		return
	}
	userID, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, err := h.messageService.ListMessages(c.Request.Context(), conversationID, userID, query.Before, query.Limit)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConversationNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": toListMessagesResponse(page)})
}
```

```text
Function: broadcastMessageCreated
Purpose: Hand a committed message to the hub for fan-out.
Parameters: the conversation id and the response already sent over HTTP.
Returns: nothing.
Called by: SendMessage, after the 201.
Why it exists: keeps the context decision and the nil-hub guard out of SendMessage.
Concurrency: runs on the request goroutine; the hub's enqueue never blocks.
Failure behavior: a nil hub is a no-op; everything else is logged inside the hub.
```

One consequence worth knowing: a retried POST with the same `client_msg_id`
returns the original message and broadcasts a second identical frame. Clients
must dedupe on message id. The dev client already does.

---

## FILE: app/internal/router/router.go
## ACTION: MODIFY

```go
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/handler"
	"github.com/l4khd4r/GuildChat/internal/ws"
)

func New(userHandler *handler.UserHandler, authHandler *handler.AuthHandler, friendshipHandler *handler.FriendshipHandler, conversationHandler *handler.ConversationHandler, messageHandler *handler.MessageHandler, wsHandler *ws.Handler, jwtManager *auth.JWTManager) *gin.Engine {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to My World!",
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})
	router.POST("/users", userHandler.CreateUser)
	router.GET("/users/:id", userHandler.GetUserByID)

	router.POST("/auth/login", authHandler.Login)

	// The socket is not in the protected group: a browser handshake cannot
	// carry an Authorization header, so ServeWS validates ?token= itself with
	// the same JWTManager the middleware uses.
	router.GET("/ws", wsHandler.ServeWS)

	protected := router.Group("/")
	protected.Use(jwtManager.Middleware())

	protected.GET("/me", userHandler.GetMe)
	protected.PUT("/me", userHandler.UpdateUser)
	protected.DELETE("/me", userHandler.DeleteMe)

	protected.POST("/users/:id/friend-request", friendshipHandler.SendFriendRequest)
	protected.POST("/friend-request/:id/accept", friendshipHandler.AcceptFriendRequest)

	protected.POST("/friend-request/:id/reject", friendshipHandler.RejectFriendRequest)
	protected.GET("/me/friends", friendshipHandler.ListFriends)
	protected.GET("/me/friend-requests", friendshipHandler.ListPendingFriendRequests)
	protected.GET("/me/friend-requests/sent", friendshipHandler.ListSentFriendRequests)

	protected.DELETE("/friend-request/:id", friendshipHandler.DeleteFriendRequest) // this is removing the row of the pending request
	protected.DELETE("/friends/:id", friendshipHandler.DeleteFriend)               // this is unfriend someone ( accepted )

	protected.POST("/conversations/dm/:id", conversationHandler.CreateDM) // :id is the *other user*; returns the existing DM if there already is one
	protected.POST("/conversations/room", conversationHandler.CreateRoom) // body: {"name": "..."}; always creates a new room, caller becomes owner

	// protected.DELETE("/conversations/:id", conversationHandler.DeleteConversationByID) // :id is the conversation id, not a user id
	protected.GET("/me/conversations", conversationHandler.ListUserConversations) // every conversation the user is a member of, DMs and rooms alike
	protected.GET("/conversations/:id", conversationHandler.GetConversation)      // one of them; 404 if it is not the caller's

	// Room membership. Both are scoped to a conversation the caller belongs to,
	// so someone outside it gets 404 rather than 403 on either.
	protected.GET("/conversations/:id/members", conversationHandler.ListMembers)              // the roster; any member may read it
	protected.POST("/conversations/:id/members", conversationHandler.AddMember)               // body: {"user_id": N}; owner/admin only, rooms only
	protected.DELETE("/conversations/:id/members/:user_id", conversationHandler.RemoveMember) // owner/admin only, rooms only

	protected.POST("/conversations/:id/messages", messageHandler.SendMessage) // body: {"body": "...", "client_msg_id": "..."}; any member may send a message
	protected.GET("/conversations/:id/messages", messageHandler.ListMessages) // ?before=<id>&limit=<n> , newest first , any member may read

	// we will neeed to implement the kick , ban ( for join is already done) , but i needed the kick and some of them we , timeout and ban are not yet implemented

	return router
}
```

---

## FILE: app/cmd/server/main.go
## ACTION: MODIFY

```go
package main

import (
	"log"
	"time"

	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/config"
	"github.com/l4khd4r/GuildChat/internal/database"
	"github.com/l4khd4r/GuildChat/internal/handler"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/router"
	"github.com/l4khd4r/GuildChat/internal/service"
	"github.com/l4khd4r/GuildChat/internal/ws"
)

func main() {
	cfg := config.Load()
	dbCfg := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	}

	// Bring the schema up to date before serving, so the app never talks to a
	// database it does not match.
	if err := database.MigrateUp(dbCfg); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Database schema is up to date")

	db, err := database.NewPostgresPool(dbCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	privateKey, err := auth.LoadPrivateKey(cfg.JWT.PrivateKeyPath)
	if err != nil {
		log.Fatalf("failed to load private key: %v", err)
	}
	publicKey, err := auth.LoadPublicKey(cfg.JWT.PublicKeyPath)
	if err != nil {
		log.Fatalf("failed to load public key: %v", err)
	}

	jwtManager := auth.NewJWTManager(privateKey, publicKey, "GuildChat", 24*time.Hour)

	log.Println("Postgres connection established")
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	friendshipRepo := repository.NewFriendshipRepository(db)
	friendshipService := service.NewFriendshipService(friendshipRepo)
	friendshipHandler := handler.NewFriendshipHandler(friendshipService)

	authService := service.NewAuthService(userRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authService)
	conversationRepo := repository.NewConversationRepository(db)
	conversationService := service.NewConversationService(conversationRepo)
	conversationHandler := handler.NewConversationHandler(conversationService)

	// The hub reads conversation rosters straight from the conversation
	// repository: membership lives in Postgres and is not mirrored in memory.
	hub := ws.NewHub(conversationRepo)
	wsHandler := ws.NewHandler(hub, jwtManager)

	messageRepo := repository.NewMessageRepository(db)
	messageService := service.NewMessageService(conversationRepo, messageRepo)
	messageHandler := handler.NewMessageHandler(messageService, hub)

	r := router.New(userHandler, authHandler, friendshipHandler, conversationHandler, messageHandler, wsHandler, jwtManager)
	log.Println("Server is running on port : " + cfg.Port)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

}
```

---

## FILE: app/internal/ws/hub_test.go
## ACTION: CREATE

```go
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/l4khd4r/GuildChat/internal/dto"
	"github.com/l4khd4r/GuildChat/internal/model"
)

// fakeMembers is a MemberLister backed by a fixed roster, so the fan-out can
// be tested without Postgres. This is the reason MemberLister is an interface.
type fakeMembers struct {
	roster map[int64][]int64 // conversation id -> member user ids
	err    error
}

func (f *fakeMembers) ListMembers(ctx context.Context, conversationID int64) ([]*model.ConversationMemberEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	entries := make([]*model.ConversationMemberEntry, 0)
	for _, userID := range f.roster[conversationID] {
		entries = append(entries, &model.ConversationMemberEntry{
			User: &model.User{ID: userID},
			Role: model.MemberMember,
		})
	}
	return entries, nil
}

// testClient builds a client with no network connection. Nothing the hub does
// touches conn: it enqueues onto a channel, and closing is a channel close.
func testClient(hub *Hub, userID int64, buffer int) *Client {
	return &Client{hub: hub, userID: userID, send: make(chan []byte, buffer)}
}

func testEvent(messageID int64, conversationID int64) Envelope {
	return NewMessageCreatedEvent(dto.MessageResponse{
		ID:             messageID,
		ConversationID: conversationID,
		Body:           "hello",
		Sender:         dto.UserResponse{ID: 1, Username: "gm"},
	})
}

func TestRegisterAddsClient(t *testing.T) {
	hub := NewHub(&fakeMembers{})
	client := testClient(hub, 1, 1)

	hub.Register(client)

	got, ok := hub.ClientFor(1)
	if !ok || got != client {
		t.Fatalf("expected user 1 to be registered")
	}
	if hub.Online() != 1 {
		t.Fatalf("expected 1 online, got %d", hub.Online())
	}
}

func TestUnregisterRemovesClient(t *testing.T) {
	hub := NewHub(&fakeMembers{})
	client := testClient(hub, 1, 1)
	hub.Register(client)

	hub.Unregister(client)

	if _, ok := hub.ClientFor(1); ok {
		t.Fatal("expected user 1 to be gone")
	}
	// Unregister closes the client, which is what stops its writePump.
	if _, open := <-client.send; open {
		t.Fatal("expected the send channel to be closed")
	}
}

func TestUnregisterIsIdempotent(t *testing.T) {
	hub := NewHub(&fakeMembers{})
	client := testClient(hub, 1, 1)
	hub.Register(client)

	// Both pumps can decide the connection is finished. A second call must not
	// panic on a double channel close.
	hub.Unregister(client)
	hub.Unregister(client)
}

func TestRegisterEvictsThePreviousConnection(t *testing.T) {
	hub := NewHub(&fakeMembers{})
	first := testClient(hub, 1, 1)
	second := testClient(hub, 1, 1)

	hub.Register(first)
	hub.Register(second)

	got, _ := hub.ClientFor(1)
	if got != second {
		t.Fatal("expected the newest connection to win")
	}
	if _, open := <-first.send; open {
		t.Fatal("expected the evicted connection to be closed")
	}
	if hub.Online() != 1 {
		t.Fatalf("expected 1 online, got %d", hub.Online())
	}
}

func TestUnregisterStaleClientKeepsTheCurrentOne(t *testing.T) {
	hub := NewHub(&fakeMembers{})
	stale := testClient(hub, 1, 1)
	current := testClient(hub, 1, 1)

	hub.Register(stale)
	hub.Register(current)

	// The evicted connection's readPump exits late and unregisters itself. It
	// must not take the reconnection down with it.
	hub.Unregister(stale)

	got, ok := hub.ClientFor(1)
	if !ok || got != current {
		t.Fatal("a late exit from the old connection evicted the new one")
	}
}

func TestBroadcastReachesMembersOnly(t *testing.T) {
	hub := NewHub(&fakeMembers{roster: map[int64][]int64{42: {1, 2}}})

	member1 := testClient(hub, 1, 4)
	member2 := testClient(hub, 2, 4)
	stranger := testClient(hub, 3, 4)
	hub.Register(member1)
	hub.Register(member2)
	hub.Register(stranger)

	hub.BroadcastToConversation(context.Background(), 42, testEvent(7, 42))

	if len(member1.send) != 1 || len(member2.send) != 1 {
		t.Fatalf("members did not both receive: %d, %d", len(member1.send), len(member2.send))
	}
	if len(stranger.send) != 0 {
		t.Fatal("a non-member received the event")
	}

	var envelope struct {
		Type string `json:"type"`
		Data struct {
			Message dto.MessageResponse `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(<-member1.send, &envelope); err != nil {
		t.Fatalf("frame did not parse: %v", err)
	}
	if envelope.Type != EventMessageCreated {
		t.Fatalf("type was %q", envelope.Type)
	}
	if envelope.Data.Message.ID != 7 || envelope.Data.Message.Body != "hello" {
		t.Fatalf("payload was %+v", envelope.Data.Message)
	}
}

func TestBroadcastSkipsOfflineMembers(t *testing.T) {
	hub := NewHub(&fakeMembers{roster: map[int64][]int64{42: {1, 2}}})

	// User 2 is a member but has no connection. That is normal, not an error.
	online := testClient(hub, 1, 4)
	hub.Register(online)

	hub.BroadcastToConversation(context.Background(), 42, testEvent(7, 42))

	if len(online.send) != 1 {
		t.Fatal("the connected member did not receive the event")
	}
}

func TestBroadcastSurvivesARosterError(t *testing.T) {
	hub := NewHub(&fakeMembers{err: errors.New("database is down")})
	client := testClient(hub, 1, 4)
	hub.Register(client)

	// Must not panic and must not deliver. The message is already committed,
	// so a failure here is logged and dropped.
	hub.BroadcastToConversation(context.Background(), 42, testEvent(7, 42))

	if len(client.send) != 0 {
		t.Fatal("delivered despite a roster failure")
	}
}

func TestBroadcastDropsASlowClient(t *testing.T) {
	hub := NewHub(&fakeMembers{roster: map[int64][]int64{42: {1}}})

	slow := testClient(hub, 1, 1) // a buffer of one, and nobody draining it
	hub.Register(slow)

	hub.BroadcastToConversation(context.Background(), 42, testEvent(7, 42)) // fills it
	hub.BroadcastToConversation(context.Background(), 42, testEvent(8, 42)) // cannot fit

	if _, ok := hub.ClientFor(1); ok {
		t.Fatal("expected the slow client to be dropped")
	}
}
```

---

## FILE: app/internal/ws/handler_test.go
## ACTION: CREATE

```go
package ws

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/dto"
)

// newTestServer builds the real handler over a throwaway RSA key, so these
// tests exercise the same JWTManager the API uses without touching keys/.
func newTestServer(t *testing.T) (*httptest.Server, *Hub, *auth.JWTManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate a key: %v", err)
	}
	jwtManager := auth.NewJWTManager(key, &key.PublicKey, "GuildChat", time.Hour)

	hub := NewHub(&fakeMembers{roster: map[int64][]int64{42: {1, 2}}})

	router := gin.New()
	router.GET("/ws", NewHandler(hub, jwtManager).ServeWS)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, hub, jwtManager
}

func socketURL(server *httptest.Server, token string) string {
	base := "ws" + strings.TrimPrefix(server.URL, "http")
	return base + "/ws?token=" + url.QueryEscape(token)
}

// waitFor polls because registration happens on the server's goroutine, which
// has no ordering guarantee with the dialer returning on ours.
func waitFor(t *testing.T, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(message)
}

func TestServeWSRejectsAMissingToken(t *testing.T) {
	server, hub, _ := newTestServer(t)

	base := "ws" + strings.TrimPrefix(server.URL, "http")
	_, response, err := websocket.DefaultDialer.Dial(base+"/ws", nil)
	if err == nil {
		t.Fatal("expected the handshake to be refused")
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
	if hub.Online() != 0 {
		t.Fatal("an unauthenticated caller was registered")
	}
}

func TestServeWSRejectsAnInvalidToken(t *testing.T) {
	server, hub, _ := newTestServer(t)

	_, response, err := websocket.DefaultDialer.Dial(socketURL(server, "not-a-jwt"), nil)
	if err == nil {
		t.Fatal("expected the handshake to be refused")
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
	if hub.Online() != 0 {
		t.Fatal("an invalid token was registered")
	}
}

func TestServeWSAcceptsAValidTokenAndUnregistersOnClose(t *testing.T) {
	server, hub, jwtManager := newTestServer(t)

	token, err := jwtManager.GenerateToken(1)
	if err != nil {
		t.Fatalf("failed to sign a token: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(socketURL(server, token), nil)
	if err != nil {
		t.Fatalf("expected the handshake to succeed: %v", err)
	}

	waitFor(t, func() bool {
		_, ok := hub.ClientFor(1)
		return ok
	}, "the connection was never registered")

	conn.Close()

	waitFor(t, func() bool { return hub.Online() == 0 }, "the connection was never unregistered")
}

func TestServeWSDeliversMessageCreated(t *testing.T) {
	server, hub, jwtManager := newTestServer(t)

	token, err := jwtManager.GenerateToken(1)
	if err != nil {
		t.Fatalf("failed to sign a token: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(socketURL(server, token), nil)
	if err != nil {
		t.Fatalf("expected the handshake to succeed: %v", err)
	}
	defer conn.Close()

	waitFor(t, func() bool {
		_, ok := hub.ClientFor(1)
		return ok
	}, "the connection was never registered")

	hub.BroadcastToConversation(t.Context(), 42, testEvent(7, 42))

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("no frame arrived: %v", err)
	}

	var envelope struct {
		Type string `json:"type"`
		Data struct {
			Message dto.MessageResponse `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("frame did not parse: %v", err)
	}
	if envelope.Type != EventMessageCreated || envelope.Data.Message.ID != 7 {
		t.Fatalf("unexpected frame: %s", payload)
	}
}
```

`t.Context()` needs Go 1.24 or newer, and `go.mod` says 1.26.5. Swap it for
`context.Background()` if that version ever drops.

---

## FILE: app/internal/handler/message_ws_db_test.go
## ACTION: CREATE

Follows the existing convention from `friendship_repository_test.go`: skipped
unless `RUN_DB_TESTS=1`, against a local Postgres. Unlike that test it builds
its own fixtures, so it does not depend on seed data.

```go
package handler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/dto"
	"github.com/l4khd4r/GuildChat/internal/model"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/service"
	"github.com/l4khd4r/GuildChat/internal/ws"
)

// TestSendMessageBroadcastsToConnectedMembers runs the whole Phase 1 path
// against a real database and real sockets:
//
//   - HTTP creation still returns 201
//   - both connected members receive message_created
//   - a connected non-member receives nothing
//   - a member who is offline does not break creation
//   - the same client_msg_id twice still returns the original message
//   - a reconnecting client recovers what it missed through GET
func TestSendMessageBroadcastsToConnectedMembers(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	db, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/guildchat")
	if err != nil {
		t.Fatalf("failed to create db pool: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	conversationRepo := repository.NewConversationRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	userService := service.NewUserService(userRepo)
	conversationService := service.NewConversationService(conversationRepo)
	messageService := service.NewMessageService(conversationRepo, messageRepo)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate a key: %v", err)
	}
	jwtManager := auth.NewJWTManager(key, &key.PublicKey, "GuildChat", time.Hour)

	hub := ws.NewHub(conversationRepo)
	wsHandler := ws.NewHandler(hub, jwtManager)
	messageHandler := NewMessageHandler(messageService, hub)

	// Three fresh users. The suffix keeps reruns from colliding on the unique
	// username and email indexes.
	suffix := time.Now().UnixNano()
	owner := createTestUser(t, ctx, userService, fmt.Sprintf("owner%d", suffix))
	member := createTestUser(t, ctx, userService, fmt.Sprintf("member%d", suffix))
	stranger := createTestUser(t, ctx, userService, fmt.Sprintf("stranger%d", suffix))

	t.Cleanup(func() {
		// The messages and membership rows go with them: both foreign keys are
		// ON DELETE CASCADE.
		for _, user := range []*model.User{owner, member, stranger} {
			_ = userService.DeleteUser(context.Background(), user.ID)
		}
	})

	room, err := conversationService.CreateRoom(ctx, owner.ID, "ws phase 1 test")
	if err != nil {
		t.Fatalf("failed to create the room: %v", err)
	}
	if err := conversationService.AddMember(ctx, owner.ID, room.ID, member.ID); err != nil {
		t.Fatalf("failed to add the member: %v", err)
	}

	router := gin.New()
	router.GET("/ws", wsHandler.ServeWS)
	protected := router.Group("/")
	protected.Use(jwtManager.Middleware())
	protected.POST("/conversations/:id/messages", messageHandler.SendMessage)
	protected.GET("/conversations/:id/messages", messageHandler.ListMessages)

	server := httptest.NewServer(router)
	defer server.Close()

	// The owner and the stranger connect. The member deliberately does not:
	// creation must work for a conversation whose other side is offline.
	ownerConn := dialSocket(t, server, jwtManager, owner.ID)
	defer ownerConn.Close()
	strangerConn := dialSocket(t, server, jwtManager, stranger.ID)
	defer strangerConn.Close()

	clientMsgID := "11111111-2222-3333-4444-" + fmt.Sprintf("%012d", suffix%1000000000000)

	first := postMessage(t, server, jwtManager, owner.ID, room.ID, "hello room", clientMsgID)

	// The sender is a member, so the sender's own socket gets the frame too.
	frame := readEnvelope(t, ownerConn)
	if frame.Type != ws.EventMessageCreated {
		t.Fatalf("unexpected event type %q", frame.Type)
	}
	if frame.Data.Message.ID != first.ID {
		t.Fatalf("pushed id %d, created id %d", frame.Data.Message.ID, first.ID)
	}
	if frame.Data.Message.Body != "hello room" {
		t.Fatalf("pushed body was %q", frame.Data.Message.Body)
	}

	// The stranger is connected but not a member. Nothing should arrive.
	strangerConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if _, _, err := strangerConn.ReadMessage(); err == nil {
		t.Fatal("a non-member received the event")
	}

	// The same client_msg_id returns the original message rather than a second
	// one. This is the idempotency the socket does not change.
	second := postMessage(t, server, jwtManager, owner.ID, room.ID, "hello room", clientMsgID)
	if second.ID != first.ID {
		t.Fatalf("retry created a second message: %d then %d", first.ID, second.ID)
	}

	// The member was offline for all of it. Reconnecting means asking HTTP,
	// which is the whole reconciliation mechanism, and the retry must not have
	// left a duplicate behind.
	page := listMessages(t, server, jwtManager, member.ID, room.ID)
	matches := 0
	for _, message := range page.Messages {
		if message.ID == first.ID {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("expected the message exactly once in the transcript, found %d", matches)
	}
}

func createTestUser(t *testing.T, ctx context.Context, users *service.UserService, name string) *model.User {
	t.Helper()
	user, err := users.CreateUser(ctx, name, name+"@example.com", "ChangeMe123!")
	if err != nil {
		t.Fatalf("failed to create %s: %v", name, err)
	}
	return user
}

func dialSocket(t *testing.T, server *httptest.Server, jwtManager *auth.JWTManager, userID int64) *websocket.Conn {
	t.Helper()
	token, err := jwtManager.GenerateToken(userID)
	if err != nil {
		t.Fatalf("failed to sign a token: %v", err)
	}
	base := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(base+"/ws?token="+url.QueryEscape(token), nil)
	if err != nil {
		t.Fatalf("failed to connect user %d: %v", userID, err)
	}
	// Give the server a moment to finish registering before anyone broadcasts.
	time.Sleep(100 * time.Millisecond)
	return conn
}

func postMessage(t *testing.T, server *httptest.Server, jwtManager *auth.JWTManager, userID int64, conversationID int64, body string, clientMsgID string) dto.MessageResponse {
	t.Helper()

	payload := fmt.Sprintf(`{"body":%q,"client_msg_id":%q}`, body, clientMsgID)
	request, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/conversations/%d/messages", server.URL, conversationID),
		strings.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+mustToken(t, jwtManager, userID))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", response.StatusCode)
	}

	var decoded struct {
		Message dto.MessageResponse `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("response did not parse: %v", err)
	}
	return decoded.Message
}

func listMessages(t *testing.T, server *httptest.Server, jwtManager *auth.JWTManager, userID int64, conversationID int64) dto.ListMessagesResponse {
	t.Helper()

	request, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/conversations/%d/messages", server.URL, conversationID), nil)
	request.Header.Set("Authorization", "Bearer "+mustToken(t, jwtManager, userID))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	var decoded struct {
		Messages dto.ListMessagesResponse `json:"messages"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("response did not parse: %v", err)
	}
	return decoded.Messages
}

func mustToken(t *testing.T, jwtManager *auth.JWTManager, userID int64) string {
	t.Helper()
	token, err := jwtManager.GenerateToken(userID)
	if err != nil {
		t.Fatalf("failed to sign a token: %v", err)
	}
	return token
}

type testEnvelope struct {
	Type string `json:"type"`
	Data struct {
		Message dto.MessageResponse `json:"message"`
	} `json:"data"`
}

func readEnvelope(t *testing.T, conn *websocket.Conn) testEnvelope {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("no frame arrived: %v", err)
	}
	var envelope testEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("frame did not parse: %v", err)
	}
	return envelope
}
```

Running the tests:

```bash
cd app && go test -race ./internal/ws/...            # no database needed
cd app && RUN_DB_TESTS=1 go test -race ./internal/handler/...
```

The `ws` tests run in the existing CI unchanged. The database one skips there,
the same way the friendship test does.

---

## FILE: docs/WEBSOCKET-PHASE-1.md
## ACTION: CREATE

````markdown
# WebSocket Phase 1

Real-time delivery of new messages. Nothing else.

## 1. Architecture

```
POST /conversations/:id/messages
        │
   MessageHandler ──► MessageService ──► MessageRepository ──► PostgreSQL
        │
        ├──► HTTP 201  {"message": MessageResponse}
        │
        └──► Hub.BroadcastToConversation
                  │
             ConversationRepository.ListMembers   (who is in this conversation)
                  │
             Hub.clients[userID]                  (who of those is connected)
                  │
             Client.send ──► writePump ──► socket
```

PostgreSQL is the source of truth for both the messages and the membership.
The hub knows one thing the database does not: which users currently have a
socket open.

## 2. HTTP versus WebSocket

| | HTTP | WebSocket |
| --- | --- | --- |
| Create a message | yes | never in Phase 1 |
| Read history | yes | no |
| Receive new messages | by polling | pushed |
| Validation, auth, idempotency | yes | not applicable |
| Guaranteed | yes, or you get an error | no, best effort |

There is no `send_message` frame. The socket is server to client only, so
every write keeps the status codes, the membership check and the
`client_msg_id` idempotency the HTTP endpoint already has.

## 3. Connection lifecycle

1. Client opens `GET /ws?token=<JWT>`.
2. `Handler.ServeWS` reads the token. Missing means 401.
3. `JWTManager.ValidateToken` runs. Invalid means 401.
4. The connection is upgraded. After this point no JSON error can be written.
5. `newClient` builds the client, `Hub.Register` puts it in the map.
6. `writePump` starts on its own goroutine; `readPump` takes over the request
   goroutine and holds the connection open.
7. On any read error or close, `readPump` calls `Hub.Unregister`, which removes
   the entry and closes the send channel, which stops `writePump`.

A user who connects twice evicts their first connection: the hub holds one
client per user id.

## 4. Message lifecycle

1. `SendMessage` validates, checks membership, and stores the message.
2. `toMessageResponse` builds one `dto.MessageResponse`.
3. That value is written as the HTTP 201.
4. `broadcastMessageCreated` passes the same value to the hub.
5. The hub reads the roster, looks up each member, and enqueues the frame for
   those who are connected.

A retried POST with the same `client_msg_id` returns the original message and
broadcasts a second identical frame. Clients dedupe on message id.

## 5. Event format

```json
{
  "type": "message_created",
  "data": {
    "message": {
      "id": 123,
      "conversation_id": 42,
      "sender": { "id": 7, "username": "rogue", "email": "r@example.com",
                  "created_at": "...", "updated_at": "..." },
      "body": "hello",
      "client_msg_id": "0d1c…",
      "created_at": "2026-09-12T04:51:00Z"
    }
  }
}
```

`data.message` is `dto.MessageResponse`, the same type the send and list
endpoints return. There is no socket-specific message shape.

Future events reuse the envelope: `typing`, `read_receipt`,
`presence_changed`. A client should ignore a `type` it does not recognise.

## 6. Hub and Client responsibilities

**Hub** owns `map[userID]*Client`, and reads conversation rosters through the
one-method `MemberLister` interface that `ConversationRepository` satisfies.
It registers, unregisters, looks up by user id, and fans out.

There is no conversation-to-clients subscription map. Membership lives in
Postgres and is read at broadcast time, so an add or a remove takes effect
immediately with no in-memory state to keep in step.

**Client** owns one connection: the socket, the user id, and a buffered channel
of outgoing frames.

## 7. Concurrency model

- Frames are marshalled once per broadcast and shared by every recipient.
- `writePump` is the only goroutine that writes to a given socket. That is what
  satisfies gorilla's one-writer rule, without holding a mutex across a network
  write.
- `readPump` is the only goroutine that reads from it.
- The hub's map is guarded by a `sync.RWMutex`: a read lock for the fan-out, a
  write lock for register and unregister.
- `enqueue` never blocks. A client whose 64-frame buffer is full is dropped, so
  one stalled reader cannot slow delivery to anyone else.
- Slow clients are collected during the fan-out and unregistered after the read
  lock is released, because `Unregister` takes the write lock.
- `Client.close` is a `sync.Once`, because both pumps can decide independently
  that the connection is finished.
- `Unregister` compares pointers before deleting, so a connection exiting late
  cannot evict the reconnection that already replaced it.

## 8. Best-effort delivery

There are no retries, no acknowledgements, no offline queues, no delivery
state. A frame that cannot be delivered is dropped and logged.

This is safe because the message is committed to PostgreSQL before any of it
runs. A client that missed something calls `GET /conversations/:id/messages`
and reconciles. That is the only recovery mechanism, and it is the same one
used for the first load.

A socket failure never affects the HTTP request. `BroadcastToConversation`
returns nothing, and it runs after the 201 has been written.

The broadcast uses a fresh context with a five second timeout rather than the
request context, because the request context is cancelled when the sender
disconnects, and a sender who posts and closes the tab must not cancel delivery
to everyone else.

## 9. Authentication

`GET /ws?token=<JWT>`, validated by the same `auth.JWTManager` as the rest of
the API. The route sits outside the protected group because a browser cannot
set an `Authorization` header on a handshake.

`CheckOrigin` accepts any origin. That is safe only because the credential is a
token in the URL rather than a cookie: a hostile page can open a socket but has
no way to obtain the victim's token.

Known limitation, deliberately not fixed in Phase 1: a JWT in a URL ends up in
access logs, proxy logs and browser history. The replacement is a short-lived
single-use ticket issued by an authenticated endpoint and exchanged here. When
that lands, `CheckOrigin` must become a real allow-list.

## 10. File map

| File | Purpose |
| --- | --- |
| `internal/ws/event.go` | The envelope, the event constant, the payload type |
| `internal/ws/client.go` | One connection, its pumps, its send buffer |
| `internal/ws/hub.go` | The user registry, the roster interface, the fan-out |
| `internal/ws/handler.go` | `GET /ws`: authenticate, upgrade, register |
| `internal/handler/message_handler.go` | Publishes after the 201 |
| `internal/router/router.go` | Registers `/ws` outside the protected group |
| `cmd/server/main.go` | Builds the hub and wires it into both |

## 11. Function map

| Function | Does |
| --- | --- |
| `NewMessageCreatedEvent` | Wraps a `MessageResponse` in its envelope |
| `Hub.Register` | Adds a client, evicting that user's previous connection |
| `Hub.Unregister` | Removes and closes a client, identity-checked |
| `Hub.ClientFor` | Finds a user's connection |
| `Hub.Online` | Counts connected users |
| `Hub.BroadcastToConversation` | Roster read, then fan-out to connected members |
| `Client.enqueue` | Non-blocking queue; false means drop this client |
| `Client.close` | Closes the send channel once, from anywhere |
| `Client.writePump` | Sole writer; drains the buffer, sends pings |
| `Client.readPump` | Sole reader; detects death and unregisters |
| `Handler.ServeWS` | Token, validate, upgrade, register, block |
| `MessageHandler.broadcastMessageCreated` | Fresh context, hands off to the hub |
````

---

## Order to apply this in

1. `go get github.com/gorilla/websocket@v1.5.3`
2. `internal/ws/event.go`, then `client.go`, `hub.go`, `handler.go`
3. `internal/handler/message_handler.go`
4. `internal/router/router.go`
5. `cmd/server/main.go`
6. `go build ./...`
7. The three test files
8. `go test -race ./internal/ws/...`
9. `docs/WEBSOCKET-PHASE-1.md`

## Not done

- None of this has been compiled, because it is not in the tree. Expect the
  compiler to catch an import pasted out of order.
- "Reconnecting and using GET retrieves missed messages" is covered at the
  transcript level in the database test, not by literally dropping and
  redialing a socket mid-test. The offline-member case already proves the same
  thing without the timing flakiness.
