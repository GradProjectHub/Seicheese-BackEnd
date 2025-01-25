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
	ctx := c.Request().Context()
	requestID := c.Response().Header().Get(echo.HeaderXRequestID)

	// トークンの取得
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		c.Logger().Error("トークンが見つかりません", "request_id", requestID)
		return echo.NewHTTPError(http.StatusUnauthorized, "トークンが必要です")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	// トークン検証
	verifiedToken, err := h.AuthClient.VerifyIDToken(ctx, token)
	if err != nil {
		c.Logger().Error("トークン検証エラー", "request_id", requestID, "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "無効なトークンです")
	}
	c.Logger().Info("トークン検証成功", "request_id", requestID, "firebase_id", verifiedToken.UID)

	// ユーザーの存在確認
	user, err := models.Users(
		qm.Where("firebase_id = ?", verifiedToken.UID),
	).One(ctx, h.DB)

	if err == sql.ErrNoRows {
		c.Logger().Info("新規ユーザー登録開始", "request_id", requestID, "firebase_id", verifiedToken.UID)
		
		tx, err := h.DB.BeginTx(ctx, nil)
		if err != nil {
			c.Logger().Error("トランザクション開始エラー", "request_id", requestID, "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "データベースエラー")
		}
		
		// トランザクションのロールバック処理
		var txErr error
		defer func() {
			if txErr != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					c.Logger().Error("ロールバックエラー", "request_id", requestID, "error", rbErr)
				}
				c.Logger().Info("トランザクションロールバック完了", "request_id", requestID)
			}
		}()

		// ユーザーの作成
		user := &models.User{
			FirebaseID: verifiedToken.UID,
			IsAdmin:    false,
		}
		
		c.Logger().Info("ユーザーレコード作成開始", "request_id", requestID)
		err = user.Insert(ctx, tx, boil.Infer())
		if err != nil {
			txErr = err
			c.Logger().Error("ユーザーレコード作成エラー", "request_id", requestID, "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー作成に失敗しました")
		}
		c.Logger().Info("ユーザーレコード作成完了", "request_id", requestID, "user_id", user.UserID)

		// ポイントレコードの作成
		point := &models.Point{
			UserID:       user.UserID,
			CurrentPoint: 0,
		}
		
		c.Logger().Info("ポイントレコード作成開始", "request_id", requestID, "user_id", user.UserID)
		err = point.Insert(ctx, tx, boil.Infer())
		if err != nil {
			txErr = err
			c.Logger().Error("ポイントレコード作成エラー", "request_id", requestID, "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の作成に失敗しました")
		}
		c.Logger().Info("ポイントレコード作成完了", "request_id", requestID)

		// トランザクションのコミット
		err = tx.Commit()
		if err != nil {
			txErr = err
			c.Logger().Error("トランザクションコミットエラー", "request_id", requestID, "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "データベースエラー")
		}
		c.Logger().Info("トランザクションコミット完了", "request_id", requestID)

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"message": "ユーザーを新規登録しました",
			"user":    user,
		})
	} else if err != nil {
		c.Logger().Error("データベースエラー", "request_id", requestID, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "データベースエラー")
	}

	c.Logger().Info("既存ユーザーのサインイン成功", "request_id", requestID, "user_id", user.UserID)
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
