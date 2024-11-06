-- +goose Up
-- +goose StatementBegin
CREATE TABLE checkin_logs (
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id INT UNSIGNED,
    seichi_id INT NOT NULL REFERENCES seichies(seichi_id) ON DELETE CASCADE,
    PRIMARY KEY (created_at, user_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE    
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE checkin_logs;
-- +goose StatementEnd
