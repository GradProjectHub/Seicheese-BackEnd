-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    user_id UNSIGNED INT AUTO_INCREMENT PRIMARY KEY,
    firebase_id VARCHAR(255) UNIQUE,
    is_admin BIT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
