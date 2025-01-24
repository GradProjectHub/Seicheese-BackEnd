-- +goose Up
-- +goose StatementBegin
CREATE TABLE markers (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    image_path VARCHAR(255) NOT NULL,
    required_points INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO markers (id, name, image_path, required_points) VALUES
('pin_1', 'Pin 1', '/static/pin_1.png', 0),
('pin_2', 'Pin 2', '/static/pin_2.png', 100),
('pin_3', 'Pin 3', '/static/pin_3.png', 200),
('pin_4', 'Pin 4', '/static/pin_4.png', 300),
('pin_5', 'Pin 5', '/static/pin_5.png', 500);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE markers;
-- +goose StatementEnd 