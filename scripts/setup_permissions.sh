#!/bin/bash

# バックアップディレクトリの設定
docker compose exec -u root go sh -c "\
mkdir -p /backup/seicheese/{database,nginx,ssl} && \
chown -R root:root /backup/seicheese && \
chmod -R 700 /backup/seicheese"

# Goアプリケーションの一時ディレクトリ設定
docker compose exec -u root go sh -c "\
mkdir -p /home/user/go/src/app/tmp && \
chown -R user:user /home/user/go/src/app/tmp"

# Nginxログディレクトリの設定
docker compose exec -u root nginx sh -c "\
mkdir -p /var/log/nginx && \
chown -R nginx:nginx /var/log/nginx && \
chmod -R 755 /var/log/nginx"

echo "Container permissions have been set successfully" 