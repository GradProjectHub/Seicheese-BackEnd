// Seicheese-Backend/src/internal/utils/version.go

package utils

import (
	"strconv"
	"strings"
)

// IsValidAppVersion アプリケーションバージョンの検証
func IsValidAppVersion(version string) bool {
	minVersionParts := []int{0, 1, 0} // 最小サポートバージョン 0.1.0

	versionParts := strings.Split(version, ".")
	if len(versionParts) != 3 {
		return false
	}

	for i, v := range versionParts {
		num, err := strconv.Atoi(v)
		if err != nil {
			return false
		}
		if num > minVersionParts[i] {
			return true
		} else if num < minVersionParts[i] {
			return false
		}
		// 等しい場合は次の桁を比較
	}
	// バージョンが完全に一致する場合はサポート対象
	return true
}
