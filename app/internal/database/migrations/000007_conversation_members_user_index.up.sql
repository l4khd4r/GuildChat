-- conversation_members is keyed on (conversation_id, user_id), and a composite
-- index is only usable left to right. That serves "who is in this conversation"
-- but not "which conversations is this user in", which has to scan the whole
-- table for want of a lookup sorted by user.
--
-- The second question is the one GET /me/conversations asks, on every app open,
-- for every user. This index is the sorted-by-user list that turns it into a
-- lookup.
CREATE INDEX idx_conversation_members_user_id
  ON conversation_members (user_id);
