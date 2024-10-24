-- +goose Up
-- +goose StatementBegin
CREATE TABLE checkin_logs (
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id INT REFERENCES users(user_id) ON DELETE CASCADE,
    seichi_id INT NOT NULL REFERENCES seichies(seichi_id) ON DELETE CASCADE,
    PRIMARY KEY (created_at, user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE checkin_logs;
-- +goose StatementEnd
