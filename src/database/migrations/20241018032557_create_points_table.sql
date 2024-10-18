-- +goose Up
-- +goose StatementBegin
CREATE TABLE points (
    firebase_id VARCHAR(255) PRIMARY KEY,
    current_point int NOT NULL,
    creaed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE points;
-- +goose StatementEnd
