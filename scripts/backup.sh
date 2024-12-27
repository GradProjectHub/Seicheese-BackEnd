#!/bin/bash

# 環境変数の読み込み
source /etc/seicheese/backup.env

# バックアップ設定
BACKUP_DIR="${BACKUP_ROOT_DIR:-/backup/seicheese}"
DATE=$(date +%Y%m%d_%H%M%S)
MYSQL_USER="${DB_USER}"
MYSQL_PASSWORD="${DB_PASSWORD}"
MYSQL_DATABASE="${DB_NAME}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"

# 環境変数チェック
if [ -z "$DB_USER" ] || [ -z "$DB_PASSWORD" ] || [ -z "$DB_NAME" ]; then
    echo "Error: Required environment variables are not set"
    exit 1
fi

# バックアップディレクトリの作成
mkdir -p "${BACKUP_DIR}/database"
mkdir -p "${BACKUP_DIR}/nginx"
mkdir -p "${BACKUP_DIR}/ssl"

# データベースのバックアップ
docker exec seicheese-db mysqldump -u${MYSQL_USER} -p${MYSQL_PASSWORD} ${MYSQL_DATABASE} > \
    "${BACKUP_DIR}/database/seicheese_db_${DATE}.sql"

# Nginx設定のバックアップ
tar -czf "${BACKUP_DIR}/nginx/nginx_conf_${DATE}.tar.gz" ./nginx/conf.d/

# SSL証明書のバックアップ
tar -czf "${BACKUP_DIR}/ssl/ssl_${DATE}.tar.gz" ./nginx/ssl/

# 古いバックアップの削除
find ${BACKUP_DIR} -type f -mtime +${RETENTION_DAYS} -delete

# バックアップの圧縮
cd ${BACKUP_DIR}
tar -czf "seicheese_full_backup_${DATE}.tar.gz" database nginx ssl

# ログ出力
echo "Backup completed at ${DATE}" >> "${BACKUP_DIR}/backup.log" 