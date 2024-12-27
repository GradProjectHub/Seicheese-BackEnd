#!/bin/bash

# SSL証明書のコピーと権限設定
if [ -d "/etc/letsencrypt/live/api.seicheese.jp" ]; then
    # nginxのssl用ディレクトリを作成
    mkdir -p nginx/ssl
    
    # 証明書をコピー
    sudo cp /etc/letsencrypt/live/api.seicheese.jp/fullchain.pem nginx/ssl/
    sudo cp /etc/letsencrypt/live/api.seicheese.jp/privkey.pem nginx/ssl/
    
    # 権限を設定
    sudo chmod 644 nginx/ssl/fullchain.pem
    sudo chmod 644 nginx/ssl/privkey.pem
    sudo chown root:root nginx/ssl/*.pem
fi

# 必要なディレクトリを作成
mkdir -p nginx/logs
mkdir -p backup/seicheese/{database,nginx,ssl}

# SSL証明書ディレクトリの権限設定
chmod 755 nginx/ssl

echo "Initial permissions have been set successfully" 