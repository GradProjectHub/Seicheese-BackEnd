-- +goose Up
-- +goose StatementBegin
CREATE TRIGGER after_user_insert
AFTER INSERT ON users
FOR EACH ROW
BEGIN
    INSERT INTO points (user_id, current_point, created_at, updated_at)
    VALUES (NEW.user_id, 0, NOW(), NOW());
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS after_user_insert;
-- +goose StatementEnd 