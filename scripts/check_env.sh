#!/bin/bash

# 必要な環境変数のリスト
REQUIRED_VARS=(
    "DB_USER"
    "DB_PASSWORD"
    "DB_NAME"
    "BACKUP_ROOT_DIR"
    "BACKUP_RETENTION_DAYS"
)

# 環境変数ファイルの存在確認
ENV_FILES=(
    "/etc/seicheese/backup.env"
    "/etc/seicheese/db.env"
)

# 環境変数ファイルの確認
for file in "${ENV_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "Error: Environment file not found: $file"
        exit 1
    fi
    
    # ファイルの権限確認
    if [ "$(stat -c %a $file)" != "600" ]; then
        echo "Warning: Incorrect permissions on $file. Setting to 600..."
        chmod 600 "$file"
    fi
done

# 環境変数の存在確認
missing_vars=()
for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -ne 0 ]; then
    echo "Error: Missing required environment variables:"
    printf '%s\n' "${missing_vars[@]}"
    exit 1
fi

echo "All environment variables are properly set" 