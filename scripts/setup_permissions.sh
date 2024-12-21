#!/bin/bash

# バックアップディレクトリの設定
BACKUP_DIR="/backup/seicheese"
mkdir -p ${BACKUP_DIR}/{database,nginx,ssl}
chown -R root:root ${BACKUP_DIR}
chmod -R 700 ${BACKUP_DIR}

# Nginxログディレクトリの設定
NGINX_LOG_DIR="/var/log/nginx"
mkdir -p ${NGINX_LOG_DIR}
chown -R nginx:nginx ${NGINX_LOG_DIR}
chmod -R 755 ${NGINX_LOG_DIR}

# Prometheusデータディレクトリの設定
PROMETHEUS_DATA_DIR="/var/lib/prometheus"
mkdir -p ${PROMETHEUS_DATA_DIR}
chown -R nobody:nogroup ${PROMETHEUS_DATA_DIR}
chmod -R 755 ${PROMETHEUS_DATA_DIR}

# Grafanaデータディレクトリの設定
GRAFANA_DATA_DIR="/var/lib/grafana"
mkdir -p ${GRAFANA_DATA_DIR}
chown -R 472:472 ${GRAFANA_DATA_DIR}  # Grafanaのデフォルトユーザー
chmod -R 755 ${GRAFANA_DATA_DIR}

# SSL証明書の権限設定
SSL_DIR="/etc/nginx/ssl"
mkdir -p ${SSL_DIR}
chown -R root:root ${SSL_DIR}
chmod -R 600 ${SSL_DIR}/*.pem
chmod 755 ${SSL_DIR}

echo "All permissions have been set successfully" 