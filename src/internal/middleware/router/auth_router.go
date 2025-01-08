// Seicheese-Backend/src/internal/router/auth_router.go

package router

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/option"
)

func NewAuthRouter(e *echo.Echo, db *sql.DB) {
	// 環境変数から現在の環境を取得
	env := os.Getenv("APP_ENV")
	var credentialsFile string

	// 環境に応じて適切な認証ファイルを選択
	switch env {
	case "development":
		credentialsFile = "configs/firebase-admin-dev.json"
	case "production":
		credentialsFile = "configs/firebase-admin-prod.json"
	default:
		credentialsFile = "configs/firebase-admin-dev.json"
	}

	// Firebase初期化
	opt := option.WithCredentialsFile(credentialsFile)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v\n", err)
	}

	// AuthClientの初期化
	authClient, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Auth client: %v\n", err)
	}

	// AuthHandlerの初期化とルートの登録
	authHandler := &handler.AuthHandler{
		DB:         db,
		AuthClient: authClient,
	}

	RegisterAuthRoutes(e, authClient, authHandler)
}

func RegisterAuthRoutes(e *echo.Echo, authClient *auth.Client, authHandler *handler.AuthHandler) {
	// ヘルスチェックエンドポイントを追加（認証不要）
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// バックエンドでのトークンとバージョンの検証エンドポイントを追加
	e.POST("/auth/validate", authHandler.ValidateToken)

	authGroup := e.Group("")
	authGroup.Use(middleware.FirebaseAuthMiddleware(authClient))
	// ハンドラーの割り当て
	authGroup.POST("/auth/signin", authHandler.SignIn)
}
