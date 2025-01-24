// Seicheese-Backend/src/internal/handler/auth.handler.go

package handler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"seicheese/internal/utils"
	"seicheese/models"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type AuthHandler struct {
	DB         *sql.DB
	AuthClient *auth.Client
}

// SignIn handler
func (h *AuthHandler) SignIn(c echo.Context) error {
	// トークンの取得
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "トークンが必要です")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// トークン検証
	verifiedToken, err := h.AuthClient.VerifyIDToken(c.Request().Context(), token)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "無効なトークンです")
	}
	log.Printf("Token verified for UID: %s", verifiedToken.UID)

	// ユーザーの存在確認
	user, err := models.Users(
		qm.Where("firebase_id = ?", verifiedToken.UID),
	).One(c.Request().Context(), h.DB)

	if err == sql.ErrNoRows {
		log.Printf("User not found for UID: %s, creating new user", verifiedToken.UID)
		// ユーザーが存在しない場合は新規登録を行う
		now := time.Now()
		newUser := models.User{
			FirebaseID: verifiedToken.UID,
			CreatedAt:  null.TimeFrom(now),
			UpdatedAt:  null.TimeFrom(now),
		}

		// トランザクション開始
		tx, err := h.DB.BeginTx(c.Request().Context(), nil)
		if err != nil {
			log.Printf("Failed to begin transaction: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "トランザクションの開始に失敗しました")
		}
		defer tx.Rollback()

		// ユーザーを保存
		err = newUser.Insert(c.Request().Context(), tx, boil.Infer())
		if err != nil {
			log.Printf("Failed to insert new user: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "ユーザーの登録に失敗しました")
		}

		// ポイントレコードを作成
		newPoint := models.Point{
			UserID:       newUser.UserID,
			CurrentPoint: 0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		err = newPoint.Insert(c.Request().Context(), tx, boil.Infer())
		if err != nil {
			log.Printf("Failed to insert new point record: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "ポイントの初期化に失敗しました")
		}

		// トランザクションをコミット
		if err := tx.Commit(); err != nil {
			log.Printf("Failed to commit transaction: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "トランザクションのコミットに失敗しました")
		}

		log.Printf("New user created successfully: %+v", newUser)

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"message": "ユーザーを新規登録しました",
			"user":    newUser,
		})
	} else if err != nil {
		log.Printf("Database error while looking up user: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "データベースエラー")
	}

	log.Printf("Existing user found: %+v", user)
	// 既存ユーザーの場合は通常のサインイン処理
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "サインインに成功しました",
		"user":    user,
	})
}

// ValidateToken ハンドラの実装
func (h *AuthHandler) ValidateToken(c echo.Context) error {
	// Authorizationヘッダーからトークンを取得
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "トークンが必要です"})
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "トークン形式が無効です"})
	}

	// トークンの検証
	token, err := h.AuthClient.VerifyIDToken(c.Request().Context(), tokenString)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "無効なトークンです"})
	}

	// 追加の検証を実行
	if err := validateToken(token); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": err.Error()})
	}

	// バージョンの検証
	var req struct {
		Version string `json:"version"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "バージョン情報が必要です"})
	}

	if !utils.IsValidAppVersion(req.Version) {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "サポートされていないアプリバージョンです"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "認証成功"})
}

// ヘルパー関数
func extractIDToken(c echo.Context) string {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// ユーザー情報の取得または作成
func (h *AuthHandler) findOrCreateUser(ctx context.Context, token *auth.Token) (*models.User, error) {
	// 既存ユーザーの検索
	user, err := models.Users(models.UserWhere.FirebaseID.EQ(token.UID)).One(ctx, h.DB)
	if err == nil {
		return user, nil
	}

	// 新規ユーザーの作成
	newUser := &models.User{
		FirebaseID: token.UID,
		CreatedAt:  null.TimeFrom(time.Now()),
	}

	if err := newUser.Insert(ctx, h.DB, boil.Infer()); err != nil {
		return nil, err
	}

	return newUser, nil
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

// トークン検証用のヘルパー関数を拡張
func validateToken(token *auth.Token) error {
	now := time.Now()

	if token == nil {
		return fmt.Errorf("token is nil")
	}

	// 有効期限の検証
	tokenExp := time.Unix(token.Expires, 0)
	if tokenExp.Before(now) {
		return fmt.Errorf("token has expired at %v", tokenExp)
	}

	// 発行時刻の検証
	tokenIat := time.Unix(token.IssuedAt, 0)
	if tokenIat.After(now) {
		return fmt.Errorf("token was issued in the future at %v", tokenIat)
	}

	// 発行者の検証
	expectedIssuer := fmt.Sprintf("https://securetoken.google.com/%s", os.Getenv("FIREBASE_PROJECT_ID"))
	if token.Issuer != expectedIssuer {
		return fmt.Errorf("invalid token issuer: expected %s, got %s", expectedIssuer, token.Issuer)
	}

	// カスタムクレームの検証
	claims := token.Claims
	if claims != nil {
		if uid, ok := claims["user_id"].(string); !ok || uid == "" {
			return fmt.Errorf("missing or invalid user_id claim")
		}

		if appVersion, ok := claims["app_version"].(string); ok {
			if !utils.IsValidAppVersion(appVersion) {
				return fmt.Errorf("unsupported app version: %s", appVersion)
			}
		}

		if role, ok := claims["role"].(string); ok {
			if !isValidRole(role) {
				return fmt.Errorf("invalid role: %s", role)
			}
		}
	}

	return nil
}
