-- +goose Up
-- +goose StatementBegin
CREATE TABLE seichies (
    seichi_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    seichi_name VARCHAR(255) NOT NULL,
    comment VARCHAR(255),
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    place_id INT NOT NULL REFERENCES places(place_id) ON DELETE CASCADE,
    content_id INT NOT NULL REFERENCES contents(content_id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE seichies;
-- +goose StatementEnd
