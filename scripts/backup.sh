#!/bin/bash

# バックアップ設定
BACKUP_DIR="/backup/seicheese"
DATE=$(date +%Y%m%d_%H%M%S)
MYSQL_USER="root"
MYSQL_PASSWORD="Wario-51"
MYSQL_DATABASE="SeiCheese"
RETENTION_DAYS=30

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