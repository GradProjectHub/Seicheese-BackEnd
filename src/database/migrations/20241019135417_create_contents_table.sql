-- +goose Up
-- +goose StatementBegin
CREATE TABLE contents (
    content_id int AUTO_INCREMENT PRIMARY KEY,
    content_name VARCHAR(255) NOT NULL UNIQUE,
    genre_id INT NOT NULL REFERENCES genres(genre_id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE contents;
-- +goose StatementEnd
