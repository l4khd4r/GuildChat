CREATE TABLE conversation_members (
  conversation_id BIGINT NOT NULL ,
  user_id BIGINT NOT NULL ,
  role VARCHAR(20) NOT NULL DEFAULT 'member' ,
  joined_at TIMESTAMP NOT NULL DEFAULT NOW(),


  PRIMARY KEY (conversation_id , user_id),

  CONSTRAINT fk_conversation_members_conversation_id
    FOREIGN KEY (conversation_id)
    REFERENCES conversations(id)
    ON DELETE CASCADE,

  CONSTRAINT conversation_members_role_check
    CHECK (role IN  ('member', 'admin', 'owner'))
);
