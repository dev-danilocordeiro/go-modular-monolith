-- +goose Up
CREATE TABLE users_users (
    id         UUID PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX users_users_email_key ON users_users (lower(email));

-- +goose Down
DROP TABLE users_users;
