-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;