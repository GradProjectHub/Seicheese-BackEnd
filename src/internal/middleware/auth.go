// Seicheese-Backend/src/internal/middleware/auth.go

package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"seicheese/internal/utils"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func FirebaseAuthMiddleware(authClient *auth.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "認証トークンがありません")
			}

			idToken := strings.TrimPrefix(token, "Bearer ")
			tokenVerified, err := authClient.VerifyIDToken(c.Request().Context(), idToken)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "無効なトークンです")
			}

			// トークンの追加検証
			if err := validateToken(tokenVerified); err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			}

			c.Set("firebase_token", idToken)
			c.Set("uid", tokenVerified.UID)
			return next(c)
		}
	}
}

// トークンの追加検証を実行
func validateToken(token *auth.Token) error {
	now := time.Now()

	if token == nil {
		return fmt.Errorf("トークンがnullです")
	}

	// 有効期限の検証
	tokenExp := time.Unix(token.Expires, 0)
	if tokenExp.Before(now) {
		return fmt.Errorf("トークンの有効期限が切れています: %v", tokenExp)
	}

	// 発行時刻の検証
	tokenIat := time.Unix(token.IssuedAt, 0)
	if tokenIat.After(now) {
		return fmt.Errorf("トークンの発行時刻が未来の日付です: %v", tokenIat)
	}

	// 発行者の検証
	expectedIssuer := fmt.Sprintf("https://securetoken.google.com/%s", os.Getenv("FIREBASE_PROJECT_ID"))
	if token.Issuer != expectedIssuer {
		return fmt.Errorf("無効なトークン発行者: 期待値 %s, 実際の値 %s", expectedIssuer, token.Issuer)
	}

	// カスタムクレームの検証
	claims := token.Claims
	if claims != nil {
		if uid, ok := claims["user_id"].(string); !ok || uid == "" {
			return fmt.Errorf("user_idクレームがないか無効です")
		}

		if appVersion, ok := claims["app_version"].(string); ok {
			if !utils.IsValidAppVersion(appVersion) {
				return fmt.Errorf("サポートされていないアプリバージョン: %s", appVersion)
			}
		}

		if role, ok := claims["role"].(string); ok {
			if !isValidRole(role) {
				return fmt.Errorf("無効な役割: %s", role)
			}
		}
	}

	return nil
}

// ユーザーロールの検証
func isValidRole(role string) bool {
	validRoles := []string{"user", "admin"}
	for _, r := range validRoles {
		if r == role {
			return true
		}
	}
	return false
}
