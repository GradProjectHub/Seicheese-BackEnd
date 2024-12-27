#!/bin/bash

# Basic認証用ディレクトリの作成
NGINX_AUTH_DIR="/etc/nginx"
mkdir -p ${NGINX_AUTH_DIR}

# ランダムパスワードの生成と環境変数での保持
ADMIN_PASSWORD=$(openssl rand -base64 12)

# htpasswdファイルの作成
# -c: 新規作成, -B: bcryptハッシュを使用
docker run --rm \
    httpd:2.4 \
    htpasswd -Bbc /tmp/.htpasswd admin "${ADMIN_PASSWORD}"

# パスワードを表示（初回ログイン用）
echo "Generated password for 'admin' user: ${ADMIN_PASSWORD}"

# htpasswdファイルの移動と権限設定
mv /tmp/.htpasswd ${NGINX_AUTH_DIR}/.htpasswd
chown root:root ${NGINX_AUTH_DIR}/.htpasswd
chmod 600 ${NGINX_AUTH_DIR}/.htpasswd

# メモリからパスワードを削除
ADMIN_PASSWORD=""

echo "Basic authentication has been set up successfully" 