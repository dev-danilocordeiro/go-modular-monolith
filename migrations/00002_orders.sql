-- +goose Up
-- user_id é só o ID textual: sem FK para users_users (fronteira entre módulos).
CREATE TABLE orders_orders (
    id          UUID PRIMARY KEY,
    user_id     TEXT        NOT NULL,
    total_cents BIGINT      NOT NULL CHECK (total_cents > 0),
    created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX orders_orders_user_id_idx ON orders_orders (user_id);

-- +goose Down
DROP TABLE orders_orders;
