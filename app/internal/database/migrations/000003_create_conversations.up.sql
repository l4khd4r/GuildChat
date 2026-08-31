CREATE TABLE conversations (
  id BIGINT PRIMARY KEY,
  type VARCHAR(255) NOT NULL,
  name VARCHAR(255) ,
  created_by BIGINT NOT NULL ,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),


  CONSTRAINT fk_conversations_created_by
    FOREIGN KEY (created_by)
    REFERENCES users(id)
    ON DELETE CASCADE,

  CONSTRAINT conversations_type_check
    CHECK (type IN ('dm', 'room'))

);
