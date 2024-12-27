#!/bin/bash

# cronジョブの追加（毎月1日の午前0時に実行）
(crontab -l 2>/dev/null; echo "0 0 1 * * cd $(pwd) && ./scripts/get_certificate.sh") | crontab -

echo "証明書の自動更新設定が完了しました" 