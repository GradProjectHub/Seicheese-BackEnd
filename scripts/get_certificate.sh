#!/bin/bash

# スクリプトがroot権限で実行されているか確認
if [ "$EUID" -ne 0 ]; then 
    echo "このスクリプトはroot権限で実行する必要があります"
    exit 1
fi

# ポート80が利用可能か確認
if netstat -tuln | grep ":80 " > /dev/null; then
    echo "ポート80が既に使用されています。証明書の取得には空いているポート80が必要です"
    exit 1
fi

# Nginxを停止（もし動いていれば）
docker compose stop nginx

# certbotを使用して証明書を取得
echo "証明書の取得を開始します..."
certbot certonly --standalone \
  -d api.seicheese.jp \
  --agree-tos \
  --email your@email.com \
  --preferred-challenges http \
  --non-interactive

# 証明書の取得が成功したら、init_permissions.shを実行
if [ $? -eq 0 ]; then
    echo "証明書の取得に成功しました。権限を設定します..."
    ./scripts/init_permissions.sh
    
    # Nginxを再起動
    docker compose start nginx
    
    echo "証明書の設定が完了しました"
else
    echo "証明書の取得に失敗しました"
    echo "詳細なログは /var/log/letsencrypt/letsencrypt.log を確認してください"
    exit 1
fi 