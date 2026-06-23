-- +goose Up
CREATE TABLE players(
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL
);

-- +goose Down

DROP TABLE players;