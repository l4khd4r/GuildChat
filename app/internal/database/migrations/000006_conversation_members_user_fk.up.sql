-- conversation_members constrained conversation_id but never user_id, so the
-- table could hold a membership for a user that does not exist. Two ways that
-- showed up:
--
--   * adding a made-up user id to a room succeeded, creating a row that every
--     roster query then silently dropped (the roster JOINs users), so the room
--     reported a member_count larger than the roster it returned;
--   * deleting a user left their memberships behind as orphans, since nothing
--     cascaded.
--
-- Clear any orphans first: the constraint cannot be added while rows violate
-- it. There is nothing to preserve in them -- they reference users that are
-- gone or never existed.
DELETE FROM conversation_members cm
WHERE NOT EXISTS (
  SELECT 1 FROM users u WHERE u.id = cm.user_id
);

ALTER TABLE conversation_members
  ADD CONSTRAINT fk_conversation_members_user_id
  FOREIGN KEY (user_id)
  REFERENCES users(id)
  ON DELETE CASCADE;
