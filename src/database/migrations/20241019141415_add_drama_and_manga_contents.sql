-- +goose Up

-- +goose StatementBegin
START TRANSACTION;
-- +goose StatementEnd

-- +goose StatementBegin
-- 管理者ユーザーの追加（存在しない場合）
INSERT IGNORE INTO users (user_id, firebase_id, is_admin) VALUES
(1, 'admin', true);
-- +goose StatementEnd

-- +goose StatementBegin
-- まずジャンルを挿入
INSERT INTO genres (genre_id, genre_name) VALUES
(1, 'アニメ'),
(2, 'マンガ'),
(3, 'ドラマ');

-- その後コンテンツを挿入
INSERT INTO contents (content_id, content_name, genre_id) VALUES
(6, '孤独のグルメ', 3),
(7, '逃げるは恥だが役に立つ', 3),
(8, 'あまちゃん', 3),
(9, 'スラムダンク', 2),
(10, 'よつばと！', 2);
-- +goose StatementEnd

-- +goose StatementBegin
-- 場所の追加
INSERT INTO places (place_id, address, zip_code) VALUES
(6, '東京都北区田端6丁目', '114-0014'),
(7, '神奈川県川崎市中原区小杉町3丁目', '211-0063'),
(8, '岩手県久慈市川崎町1-1', '028-0051'),
(9, '神奈川県鎌倉市由比ガ浜4丁目', '248-0014'),
(10, '埼玉県春日部市', '344-0067');
-- +goose StatementEnd

-- +goose StatementBegin
-- 聖地データの追加
INSERT INTO seichies (user_id, seichi_name, comment, latitude, longitude, place_id, content_id) VALUES
(1, '田端銀座商店街', '五郎さんが訪れた飲食店が多数存在する商店街', 35.7384, 139.7610, 6, 6),
(1, '武蔵小杉駅周辺', 'みくりとつばさが働いていたオフィス街', 35.5766, 139.6599, 7, 7),
(1, '久慈市小袖海女センター', 'ドラマの舞台となった北三陸の実際のモデル地', 40.1859, 141.8034, 8, 8),
(1, '由比ガ浜', '湘北高校のモデルとされる場所近くの海岸', 35.3086, 139.5505, 9, 9),
(1, '春日部駅周辺', 'よつばたちが暮らす街のモデル', 35.9750, 139.7521, 10, 10);
-- +goose StatementEnd

-- +goose StatementBegin
COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM seichies WHERE content_id IN (SELECT content_id FROM contents WHERE content_name IN ('孤独のグルメ', '逃げるは恥だが役に立つ', 'あまちゃん', 'スラムダンク', 'よつばと！'));
DELETE FROM places WHERE place_id IN (6, 7, 8, 9, 10);
DELETE FROM contents WHERE content_id IN (6, 7, 8, 9, 10);
DELETE FROM users WHERE user_id = 1 AND firebase_id = 'admin';
DELETE FROM genres WHERE genre_id IN (1, 2, 3);
-- +goose StatementEnd 