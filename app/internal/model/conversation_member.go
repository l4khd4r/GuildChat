package model

import "time"

const (
	ConversationDM   = "dm"
	ConversationRoom = "room"

	MemberOwner  = "owner"
	MemberAdmin  = "admin"
	MemberMember = "member"
)

// CanManageMembers reports whether a role may add other members.
//
// Roles form a ladder -- owner > admin > member -- and this is the first rung
// that has any teeth. Keeping the rule in one named function means the
// permission is stated once; the endpoints ask the question rather than
// spelling out which roles qualify, so widening or narrowing it later is a
// single edit here.
func CanManageMembers(role string) bool {
	return role == MemberOwner || role == MemberAdmin
}

// RoleRank puts the ladder in numbers so that permissions can be compared
// rather than enumerated. An unknown role ranks 0, below every real one, so a
// role this code does not recognise can never authorise anything.
func RoleRank(role string) int {
	switch role {
	case MemberOwner:
		return 3
	case MemberAdmin:
		return 2
	case MemberMember:
		return 1
	default:
		return 0
	}
}

// CanRemove reports whether someone with removerRole may remove someone with
// targetRole.
//
// The rule is strictly greater, which gives the whole policy in one line:
//
//	owner  removes admins and members
//	admin  removes members, but not other admins and not the owner
//	member removes nobody
//
// Strictness is what protects peers from each other -- an admin cannot demote
// the room out from under a fellow admin. It deliberately says nothing about
// removing *yourself*: leaving is a different question with a different rule,
// and callers must check for that case before asking this one.
func CanRemove(removerRole string, targetRole string) bool {
	return RoleRank(removerRole) > RoleRank(targetRole)
}

type ConversationMember struct {
	ConversationID int64     `json:"conversation_id"`
	UserID         int64     `json:"user_id"`
	Role           string    `json:"role"`
	JoinedAt       time.Time `json:"joined_at"`
}

// ConversationAccess is the answer to "may this user do things here, and what
// things": the conversation's type, and the caller's role in it.
//
// Obtaining one is itself the membership check -- there is no such value for a
// user who is not a member, so an endpoint that holds one has already proved
// the caller belongs.
type ConversationAccess struct {
	ConversationType string
	Role             string
}

// ConversationMemberEntry is one row of a conversation's roster: who they are,
// what they may do, and since when.
//
// It pairs the membership with the whole user because a roster is unreadable
// without names -- a list of user ids would force the client into an N+1 of its
// own, which is the same reason ConversationListEntry embeds the other user.
type ConversationMemberEntry struct {
	User     *User
	Role     string
	JoinedAt time.Time
}
