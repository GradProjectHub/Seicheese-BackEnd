-- +goose Up
-- +goose StatementBegin
CREATE TABLE points (
    user_id INT PRIMARY KEY REFERENCES users(user_id),
    current_point INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE points;
-- +goose StatementEnd
