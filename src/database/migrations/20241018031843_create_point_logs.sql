-- +goose Up
-- +goose StatementBegin
CREATE TABLE point_logs (
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    firebase_id VARCHAR(255) REFERENCES users(firebase_id) ON DELETE CASCADE,
    point INT NOT NULL,
    PRIMARY KEY (created_at, firebase_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE point_logs;
-- +goose StatementEnd
