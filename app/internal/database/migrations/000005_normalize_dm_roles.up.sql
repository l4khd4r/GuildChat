-- Both sides of a DM are plain members. "owner" is a room concept, and roles
-- only became load-bearing with the member endpoints: before that, the role a
-- DM participant carried was never read, so the asymmetry was invisible.
--
-- GetOrCreateDM used to give the initiator 'owner' and the other side
-- 'member'. Left alone, whoever happened to open a DM would be able to
-- administer a conversation between equals as soon as any rule consults
-- CanManageMembers.
UPDATE conversation_members cm
SET role = 'member'
FROM conversations c
WHERE c.id = cm.conversation_id
  AND c.type = 'dm'
  AND cm.role <> 'member';
