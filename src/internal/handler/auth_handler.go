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
	UserHandler *UserHandler
}

func NewAuthHandler(db *sql.DB, authClient *auth.Client) *AuthHandler {
	if db == nil {
		log.Fatal("Database connection is nil")
	}
	if authClient == nil {
		log.Fatal("Auth client is nil")
	}
	userHandler := NewUserHandler(db)
	if userHandler == nil {
		log.Fatal("Failed to create UserHandler")
	}
	return &AuthHandler{
		DB:          db,
		AuthClient:  authClient,
		UserHandler: userHandler,
	}
}

// ポイント情報作成用の関数
func (h *AuthHandler) createInitialPoint(ctx context.Context, tx *sql.Tx, user *models.User) error {
	log.Printf("ポイントレコード作成開始: user_id=%d", user.UserID)
	
	point := &models.Point{
		UserID:       user.UserID,
		CurrentPoint: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	if err := point.Insert(ctx, tx, boil.Infer()); err != nil {
		log.Printf("ポイントレコード作成エラー: %v", err)
		return fmt.Errorf("failed to create point record: %v", err)
	}
	
	log.Printf("ポイントレコード作成完了: user_id=%d", user.UserID)
	return nil
}

// RegisterUser handles new user registration
func (h *AuthHandler) RegisterUser(c echo.Context) error {
	ctx := c.Request().Context()
	log.Printf("新規ユーザー登録処理開始")

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

	// ユーザーの存在確認
	exists, err := models.Users(
		models.UserWhere.FirebaseID.EQ(verifiedToken.UID),
	).Exists(ctx, h.DB)
	if err != nil {
		log.Printf("ユーザー存在確認エラー: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の確認に失敗しました")
	}
	if exists {
		log.Printf("ユーザーが既に存在します: firebase_id=%s", verifiedToken.UID)
		return echo.NewHTTPError(http.StatusConflict, "既に登録されているユーザーです")
	}

	// ユーザー情報の取得または作成
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(verifiedToken.UID),
	).One(ctx, h.DB)

	var point models.Point
	if err == sql.ErrNoRows {
		log.Printf("新規ユーザーとして処理開始: firebase_id=%s", verifiedToken.UID)

		// 新規ユーザーの作成
		now := time.Now()
		user = &models.User{
			FirebaseID: verifiedToken.UID,
			CreatedAt:  null.TimeFrom(now),
			UpdatedAt:  null.TimeFrom(now),
		}

		log.Printf("新規ユーザー作成試行: firebase_id=%s", verifiedToken.UID)

		if err := user.Insert(ctx, h.DB, boil.Infer()); err != nil {
			log.Printf("ユーザー作成エラー: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "ユーザーの登録に失敗しました")
		}

		log.Printf("ユーザーを作成しました: user_id=%d, firebase_id=%s", user.UserID, user.FirebaseID)

		// ポイント情報の作成
		point = models.Point{
			UserID:       user.UserID,
			CurrentPoint: 0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		log.Printf("ポイント情報作成試行: user_id=%d", user.UserID)

		if err := point.Insert(ctx, h.DB, boil.Infer()); err != nil {
			log.Printf("ポイント情報作成エラー: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の作成に失敗しました")
		}

		log.Printf("ポイント情報を作成しました: user_id=%d", user.UserID)

	} else if err != nil {
		log.Printf("ユーザー情報の取得に失敗: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗しました")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "ユーザーを新規登録しました",
		"user": map[string]interface{}{
			"user_id":    user.UserID,
			"created_at": user.CreatedAt.Time,
		},
		"point": point,
	})
}

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

	// ユーザー情報の取得または作成
	user, created, err := h.UserHandler.GetOrCreateUser(ctx, verifiedToken.UID)
	if err != nil {
		log.Printf("ユーザー取得または作成エラー: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗しました")
	}

	if created {
		log.Printf("新規ユーザーを作成しました: user_id=%d", user.UserID)
	} else {
		log.Printf("既存ユーザーを取得しました: user_id=%d", user.UserID)
	}

	// ポイント情報の取得
	point, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(ctx, h.DB)
	if err != nil {
		log.Printf("ポイント情報の取得に失敗: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の取得に失敗しました")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "サインインに成功しました",
		"user": map[string]interface{}{
			"user_id":    user.UserID,
			"created_at": user.CreatedAt.Time,
		},
		"point": point,
	})
}

// GetCurrentUser handles getting the current user's information
func (h *AuthHandler) GetCurrentUser(c echo.Context) error {
	ctx := c.Request().Context()
	uid := c.Get("uid").(string)

	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, h.DB)
	if err != nil {
		log.Printf("ユーザー情報の取得に失敗: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗しました")
	}

	point, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(ctx, h.DB)
	if err != nil {
		log.Printf("ポイント情報の取得に失敗: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の取得に失敗しました")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"user_id":    user.UserID,
			"created_at": user.CreatedAt.Time,
		},
		"point": point,
	})
}

// UpdateUser handles updating the current user's information
func (h *AuthHandler) UpdateUser(c echo.Context) error {
	// TODO: Implement user update logic
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"message": "未実装の機能です",
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

	if err != nil && err != sql.ErrNoRows {
		log.Printf("ユーザー検索エラー: %v", err)
		return nil, nil, false, fmt.Errorf("ユーザー検索エラー: %v", err)
	}

	if err == sql.ErrNoRows {
		log.Printf("新規ユーザーとして処理開始: firebase_id=%s", token.UID)
	} else {
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

	// トランザクション開始
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("トランザクション開始エラー: %v", err)
		return nil, nil, false, fmt.Errorf("トランザクション開始に失敗しました: %v", err)
	}

	// トランザクションのロールバック処理
	var txErr error
	defer func() {
		if tx != nil && txErr != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("トランザクションのロールバックに失敗: %v", rbErr)
			} else {
				log.Printf("トランザクションをロールバックしました")
			}
		}
	}()

	log.Printf("新規ユーザー作成開始: firebase_id=%s", token.UID)

	// 新規ユーザーの作成
	now := time.Now()
	newUser := &models.User{
		FirebaseID: token.UID,
		CreatedAt:  null.TimeFrom(now),
		UpdatedAt:  null.TimeFrom(now),
	}

	log.Printf("新規ユーザー作成試行: firebase_id=%s", newUser.FirebaseID)

	if err := newUser.Insert(ctx, tx, boil.Infer()); err != nil {
		txErr = err
		log.Printf("ユーザー作成エラー: %v, firebase_id=%s", err, token.UID)
		return nil, nil, false, fmt.Errorf("ユーザー作成エラー: %v", err)
	}

	log.Printf("新規ユーザー作成SQL完了: user_id=%d, firebase_id=%s", newUser.UserID, newUser.FirebaseID)

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		txErr = err
		log.Printf("トランザクションのコミットに失敗: %v, firebase_id=%s", err, token.UID)
		return nil, nil, false, fmt.Errorf("トランザクションのコミットに失敗: %v", err)
	}
	tx = nil

	log.Printf("トランザクションをコミットしました")

	// トリガーの実行を待機
	time.Sleep(100 * time.Millisecond)

	// ポイント情報の取得を複数回試行
	var point *models.Point
	var pointErr error
	for i := 0; i < 5; i++ {
		point, pointErr = models.Points(
		qm.Where("user_id = ?", newUser.UserID),
		).One(ctx, h.DB)
		if pointErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if pointErr != nil {
		log.Printf("ポイント情報の取得に失敗: %v", pointErr)
		return nil, nil, false, fmt.Errorf("ポイント情報の取得に失敗: %v", pointErr)
	}

	log.Printf("新規ユーザーのポイント情報取得完了: user_id=%d", newUser.UserID)
	return newUser, point, true, nil
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

func (h *AuthHandler) SignOut(c echo.Context) error {
	// サインアウト処理の実装
	log.Printf("サインアウト処理開始")
	// 必要な処理を追加
	return c.JSON(http.StatusOK, map[string]string{"message": "サインアウトしました"})
}

func (h *AuthHandler) DeleteUser(c echo.Context) error {
	// ユーザー削除処理の実装
	log.Printf("ユーザー削除処理開始")
	// 必要な処理を追加
	return c.JSON(http.StatusOK, map[string]string{"message": "ユーザーが削除されました"})
}

func (h *AuthHandler) GetUserPoints(c echo.Context) error {
	// ユーザーのポイント情報取得処理の実装
	log.Printf("ユーザーのポイント情報取得処理開始")
	// 必要な処理を追加
	return c.JSON(http.StatusOK, map[string]interface{}{"points": 0}) // 例として0を返す
}
