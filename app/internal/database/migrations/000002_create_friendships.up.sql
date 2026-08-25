CREATE TABLE friendships (
    id BIGSERIAL PRIMARY KEY,

    requester_id BIGINT NOT NULL,
    receiver_id BIGINT NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_friendship_requester
        FOREIGN KEY (requester_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_friendship_receiver
        FOREIGN KEY (receiver_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT friendship_users_different
        CHECK (requester_id <> receiver_id),

    CONSTRAINT friendship_status_valid
        CHECK (status IN ('pending', 'accepted', 'rejected'))
);


CREATE UNIQUE INDEX unique_friendship_pair
ON friendships (
    LEAST(requester_id, receiver_id),
    GREATEST(requester_id, receiver_id)
);
