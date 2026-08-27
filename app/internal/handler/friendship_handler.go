package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/l4khd4r/GuildChat/internal/auth"
	"github.com/l4khd4r/GuildChat/internal/repository"
	"github.com/l4khd4r/GuildChat/internal/service"
)

// FriendshipHandler serves the friendship endpoints. Its job is only HTTP:
// read the request (path params, the authenticated user id), call the service,
// translate the outcome into a status code, and map the returned model to a
// dto before writing it. No business rules live here, and no model type is
// ever passed to c.JSON directly (see mapper.go for why).
//
// Routes, all behind the JWT middleware (see internal/router/router.go):
//
//	POST   /users/:id/friend-request     :id = the user being asked
//	POST   /friend-request/:id/accept    :id = the friendship
//	POST   /friend-request/:id/reject    :id = the friendship
//	DELETE /friend-request/:id           :id = the friendship
//	GET    /me/friends
//	GET    /me/friend-requests           requests I received
//	GET    /me/friend-requests/sent      requests I sent
//
// Note the two different meanings of :id. Sending is addressed by *user*
// (you know who you want to befriend, the friendship does not exist yet);
// answering is addressed by *friendship* (the row already exists, and its id
// is what the list endpoints hand you).
type FriendshipHandler struct {
	friendshipService *service.FriendshipService
}

func NewFriendshipHandler(friendshipService *service.FriendshipService) *FriendshipHandler {
	return &FriendshipHandler{
		friendshipService: friendshipService,
	}
}

// SendFriendRequest handles POST /users/:id/friend-request.
//
// The requester is the authenticated user, the receiver is :id. Responses:
//
//	201 the created friendship, status "pending"
//	400 :id is not a number, or you addressed yourself
//	409 these two are already linked, whatever the status of that link
//	500 anything else
//
// 409 also covers the race where two users send to each other at the same
// instant: the loser hits the unique_friendship_pair index and the repository
// reports it as ErrFriendshipAlreadyExists rather than a raw driver error.
func (h *FriendshipHandler) SendFriendRequest(c *gin.Context) {

	receiverId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid receiver ID"})
		return
	}

	requesterId, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendship, err := h.friendshipService.SendFriendRequest(c.Request.Context(), requesterId, receiverId)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrCannotFriendYourself):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		case errors.Is(err, repository.ErrFriendshipAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			// Sentinel errors are safe to show; anything else is logged and
			// answered with a generic message, so internal details (SQL,
			// driver text) do not leak to the client.
			log.Printf("send friend request %d -> %d: %v", requesterId, receiverId, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send friend request"})

		}
		return
	}
	c.JSON(http.StatusCreated, toFriendshipResponse(friendship))
}

// AcceptFriendRequest handles POST /friend-request/:id/accept.
//
// :id is the friendship id, taken from the pending list. Responses:
//
//	200 the friendship, now "accepted"
//	400 :id is not a number
//	404 no such request, you are not its receiver, or it is not pending
//	500 anything else
//
// The three 404 cases are deliberately indistinguishable. The authorisation
// check lives in the SQL: the UPDATE matches on receiver_id = the caller, so a
// user cannot accept a request that was not addressed to them, and no row is
// updated if they try. Telling them apart would leak the existence of other
// users' requests.
func (h *FriendshipHandler) AcceptFriendRequest(c *gin.Context) {
	friendshipId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship ID"})
		return
	}

	receiverId, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendship, err := h.friendshipService.AcceptFriendRequest(c.Request.Context(), friendshipId, receiverId)
	if err != nil {
		if errors.Is(err, repository.ErrFriendshipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "friend request not found"})
			return
		}

		log.Printf("accept friend request %d by %d: %v", friendshipId, receiverId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept friend request"})
		return
	}
	c.JSON(http.StatusOK, toFriendshipResponse(friendship))
}

// RejectFriendRequest handles POST /friend-request/:id/reject.
//
// Same shape and same authorisation rule as AcceptFriendRequest. Responses:
//
//	200 the rejected friendship, status "rejected"
//	400 :id is not a number
//	404 no such request, you are not its receiver, or it is not pending
//	500 anything else
//
// Rejecting deletes the row rather than marking it (see the Reject method in
// the repository for why), so the returned body describes something that no
// longer exists. That is intentional: the client gets confirmation of what it
// just removed, and the pair is free to try again later.
func (h *FriendshipHandler) RejectFriendRequest(c *gin.Context) {
	friendshipId, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship ID"})
		return
	}

	receiverId, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendship, err := h.friendshipService.RejectFriendRequest(c.Request.Context(), friendshipId, receiverId)
	if err != nil {
		if errors.Is(err, repository.ErrFriendshipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "friend request not found"})
			return
		}

		log.Printf("reject friend request %d by %d: %v", friendshipId, receiverId, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject friend request"})
		return
	}
	c.JSON(http.StatusOK, toFriendshipResponse(friendship))
}

// ListFriends handles GET /me/friends.
//
// Returns the users the caller is actually friends with, i.e. accepted
// friendships in either direction: the query resolves "the other user" with a
// CASE, so it does not matter who originally sent the request. Responses:
//
//	200 a (possibly empty) array of users
//	500 anything else
//
// The response is []dto.UserResponse, not []model.User. Both would serialise
// to the same JSON today, but only the dto keeps doing so after someone adds a
// field to the model.
func (h *FriendshipHandler) ListFriends(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	friends, err := h.friendshipService.ListFriends(c.Request.Context(), userID)
	if err != nil {
		log.Printf("list friends of %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list friends"})
		return
	}

	c.JSON(http.StatusOK, toUserResponses(friends))
}

// ListPendingFriendRequests handles GET /me/friend-requests.
//
// The caller's inbox: requests other people sent to them and that are still
// pending. Each entry carries the friendship id, so the client can POST it to
// /friend-request/:id/accept or /reject, and the user who sent it. Responses:
//
//	200 a (possibly empty) array of friend requests
//	500 anything else
//
// Compare with ListSentFriendRequests below: same shape, opposite direction.
func (h *FriendshipHandler) ListPendingFriendRequests(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	pendingRequests, err := h.friendshipService.ListPendingRequests(c.Request.Context(), userID)
	if err != nil {
		log.Printf("list pending friend requests of %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list friend requests"})
		return
	}

	c.JSON(http.StatusOK, toFriendRequestResponses(pendingRequests))
}

// ListSentFriendRequests handles GET /me/friend-requests/sent.
//
// The caller's outbox: requests they sent that nobody has answered yet. The
// embedded user is the receiver, i.e. who they are waiting on. Responses:
//
//	200 a (possibly empty) array of friend requests
//	500 anything else
//
// Rejected requests never appear here, because rejecting deletes the row.
// A request that vanishes from this list was either accepted (the user shows
// up in /me/friends) or rejected.
func (h *FriendshipHandler) ListSentFriendRequests(c *gin.Context) {

	senderID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sentRequests, err := h.friendshipService.ListSentRequests(c.Request.Context(), senderID)

	if err != nil {
		log.Printf("list sent friend requests of %d: %v", senderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sent friend requests"})
		return
	}

	c.JSON(http.StatusOK, toFriendRequestResponses(sentRequests))
}

// DeleteFriendRequest handles DELETE /friend-request/:id.
//
// Removes a pending request. Both parties can: the sender cancels one they no
// longer want, the receiver dismisses one they would rather not answer.
// Responses:
//
//	200 a confirmation message
//	400 :id is not a number
//	404 no pending request with that id involving the caller
//	500 anything else
//
// The 404 covers three cases the caller cannot tell apart, and should not be
// able to: the id does not exist, it belongs to two other users, or the request
// has already been answered. Saying which would leak whether a friendship
// exists between people the caller has nothing to do with.
//
// For the receiver this overlaps with POST /friend-request/:id/reject - both
// delete the row. Reject answers with the friendship, this answers with a
// message.
func (h *FriendshipHandler) DeleteFriendRequest(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendshipID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship ID"})
		return
	}

	err = h.friendshipService.DeleteFriendRequest(c.Request.Context(), friendshipID, userID)

	if err != nil {
		if errors.Is(err, repository.ErrFriendshipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "friend request not found"})
			return
		}

		log.Printf("delete friend request %d by %d: %v", friendshipID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete friend request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "friend request deleted successfully"})
}

func (h *FriendshipHandler) DeleteFriend(c *gin.Context) {
	userID, ok := auth.GetUserIDFromContext(c)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	friendshipID, err := strconv.ParseInt(c.Param("id"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship ID"})
		return
	}

	err = h.friendshipService.DeleteFriend(c.Request.Context(), friendshipID, userID)

	if err != nil {
		if errors.Is(err, repository.ErrFriendshipNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "friendship not found"})
			return
		}

		log.Printf("delete friendship %d by %d: %v", friendshipID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete friendship"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "friendship deleted successfully"})
}
