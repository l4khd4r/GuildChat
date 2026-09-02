-- Drops the constraint only. The orphan rows deleted by the up migration are
-- not restored: they referenced users that do not exist, so there is nothing
-- to restore them to.
ALTER TABLE conversation_members
  DROP CONSTRAINT fk_conversation_members_user_id;
