#!/bin/bash

set -e

# データベースの接続情報 (docker-compose.ymlと一致させる)
DB_USER="root"
DB_PASS="Wario-51"
DB_HOST="db"
DB_PORT=3306
DB_NAME="SeiCheese"

# gooseの設定 (コンテナ内のパス)
MIGRATIONS_DIR="/home/user/go/src/app/database/migrations"


echo "Running migrations..."

# gooseを実行
docker compose exec go bash -c "goose -dir ${MIGRATIONS_DIR} mysql \"${DB_USER}:${DB_PASS}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}?parseTime=true\" up"


echo "Migrations complete. Generating code..."

# sqlboilerを実行
docker compose exec go sqlboiler mysql --wipe --no-tests

echo "Code generation complete."
