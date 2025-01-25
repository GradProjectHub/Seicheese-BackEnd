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
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type AuthHandler struct {
	DB         *sql.DB
	AuthClient *auth.Client
}

// ポイント情報作成用の関数
func (h *AuthHandler) createInitialPoint(ctx context.Context, tx *sql.Tx, user *models.User) error {
	log.Printf("ポイントレコード作成開始: user_id=%d", user.UserID)

	// UserHandlerを作成
	userHandler := &UserHandler{DB: h.DB}

	// UserHandlerのメソッドを呼び出してポイントを作成
	if err := userHandler.CreateInitialPoint(ctx, tx, user); err != nil {
		log.Printf("ポイントレコード作成エラー: %v", err)
		return fmt.Errorf("failed to create point record: %v", err)
	}

	log.Printf("ポイントレコード作成完了: user_id=%d", user.UserID)
	return nil
}

// SignIn handler
func (h *AuthHandler) SignIn(c echo.Context) error {
	ctx := c.Request().Context()
	log.Printf("サインイン処理開始")

	// トークンの取得と検証
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		log.Printf("トークンが見つかりません")
		return echo.NewHTTPError(http.StatusUnauthorized, "トークンが必要です")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	verifiedToken, err := h.AuthClient.VerifyIDToken(ctx, token)
	if err != nil {
		log.Printf("トークン検証エラー: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "無効なトークンです")
	}
	log.Printf("トークン検証成功: firebase_id=%s", verifiedToken.UID)

	// ユーザーの取得または作成
	log.Printf("ユーザーの取得または作成を開始: firebase_id=%s", verifiedToken.UID)
	user, point, isNew, err := h.findOrCreateUser(ctx, verifiedToken)
	if err != nil {
		log.Printf("ユーザー取得/作成エラー: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の処理に失敗しました")
	}
	log.Printf("ユーザーの取得または作成が完了: user_id=%d, isNew=%v", user.UserID, isNew)

	if isNew {
		log.Printf("新規ユーザー登録完了: user_id=%d, firebase_id=%s", user.UserID, user.FirebaseID)
		return c.JSON(http.StatusCreated, map[string]interface{}{
			"message": "ユーザーを新規登録しました",
			"user":    user,
			"point":   point,
		})
	}

	log.Printf("既存ユーザーのサインイン完了: user_id=%d, firebase_id=%s", user.UserID, user.FirebaseID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "サインインに成功しました",
		"user":    user,
		"point":   point,
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
func (h *AuthHandler) findOrCreateUser(ctx context.Context, token *auth.Token) (*models.User, *models.Point, bool, error) {
	log.Printf("ユーザー検索開始: firebase_id=%s", token.UID)

	// 既存ユーザーの検索
	existingUser, err := models.Users(
		qm.Where("firebase_id = ?", token.UID),
	).One(ctx, h.DB)

	if err == nil {
		log.Printf("既存ユーザーを検出: firebase_id=%s, user_id=%d", token.UID, existingUser.UserID)
		// 既存ユーザーのポイント情報を取得
		point, err := models.Points(
			qm.Where("user_id = ?", existingUser.UserID),
		).One(ctx, h.DB)
		if err != nil {
			log.Printf("ポイント情報の取得に失敗: %v", err)
			return nil, nil, false, fmt.Errorf("ポイント情報の取得に失敗: %v", err)
		}
		return existingUser, point, false, nil
	}

	if err != sql.ErrNoRows {
		log.Printf("ユーザー検索エラー: %v", err)
		return nil, nil, false, fmt.Errorf("ユーザー検索エラー: %v", err)
	}

	log.Printf("新規ユーザー作成開始: firebase_id=%s", token.UID)

	// ユーザー登録を呼び出す
	userHandler := &UserHandler{DB: h.DB}
	req := echo.NewContext(c.Request(), c.Response())
	req.Set("uid", token.UID)
	user, point, isNew, err := userHandler.RegisterUser(req)
	if err != nil {
		log.Printf("ユーザー登録エラー: %v", err)
		return nil, nil, false, fmt.Errorf("ユーザー登録エラー: %v", err)
	}

	log.Printf("ユーザーの取得または作成が完了: user_id=%d, isNew=%v", user.UserID, isNew)
	return user, point, isNew, nil
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
