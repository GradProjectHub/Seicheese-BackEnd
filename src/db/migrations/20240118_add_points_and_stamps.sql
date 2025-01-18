-- ユーザーテーブルの変更
ALTER TABLE users
ADD COLUMN points INT NOT NULL DEFAULT 0,
ADD COLUMN stamps JSON;

-- チェックインログテーブルの変更
ALTER TABLE checkin_logs
ADD COLUMN points INT NOT NULL DEFAULT 0,
ADD COLUMN stamp_id INT; 