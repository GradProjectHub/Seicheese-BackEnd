-- +goose Up

-- +goose StatementBegin
START TRANSACTION;
-- +goose StatementEnd

-- +goose StatementBegin
-- アニメ作品の追加
INSERT INTO contents (content_id, content_name, genre_id) VALUES
(1, '君の名は。', 1),
(2, 'けいおん！', 1),
(3, 'らき☆すた', 1),
(4, 'ゆるキャン△', 1),
(5, '氷菓', 1),
(6, 'チェックイン用', 3);
-- +goose StatementEnd

-- +goose StatementBegin
-- 場所の追加
INSERT INTO places (place_id, address, zip_code) VALUES
(1, '東京都新宿区須賀町1', '160-0017'),
(2, '滋賀県豊郷町石畑518', '529-1202'),
(3, '埼玉県久喜市鷲宮1丁目3-1', '340-0217'),
(4, '山梨県南都留郡富士河口湖町浅川1163-1', '401-0301'),
(5, '岐阜県高山市上三之町', '506-0846'),
(6, '愛知県名古屋市熱田区神宮4丁目7-35', '456-0031');
-- +goose StatementEnd

-- +goose StatementBegin
-- 聖地データの追加
INSERT INTO seichies (user_id, seichi_name, comment, latitude, longitude, place_id, content_id) VALUES
(1, '須賀神社', '映画のクライマックスシーンの舞台となった神社', 35.6858, 139.7077, 1, 1),
(1, '豊郷小学校旧校舎群', '劇中の学校のモデルとなった建物', 35.2078, 136.2307, 2, 2),
(1, '鷲宮神社', '鷹宮神社のモデルとなった神社', 36.0907, 139.6728, 3, 3),
(1, '浅間神社', 'リンたちが初詣に訪れた神社', 35.4778, 138.7553, 4, 4),
(1, '飛騨高山', '古川町を中心とした高山市の街並み', 36.1408, 137.2597, 5, 5),
(1, '名古屋工学院専門学校3号館', '名古屋工学院専門学校の3号館建物です。', 35.1209584, 136.911830, 6, 6);
-- +goose StatementEnd

-- +goose StatementBegin
COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM seichies WHERE content_id IN (SELECT content_id FROM contents WHERE content_name IN ('君の名は。', 'けいおん！', 'らき☆すた', 'ゆるキャン△', '氷菓', 'チェックイン用'));
DELETE FROM places WHERE place_id IN (1, 2, 3, 4, 5,6);
DELETE FROM contents WHERE content_id IN (1, 2, 3, 4, 5,6);
-- +goose StatementEnd 