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
	// 環境変数から認証情報ファイルのパスを取得
	credPath := os.Getenv("FIREBASE_SDK_PATH")
	if credPath == "" {
		log.Fatalf("FIREBASE_SDK_PATHの環境変数が設定されていません")
	}

	// Firebase初期化
	opt := option.WithCredentialsFile(credPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v\n", err)
	}

	// AuthClientの初期化
	authClient, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Auth client: %v\n", err)
	}

	// 認証ミドルウェアの初期化
	authMiddleware := middleware.NewAuthMiddleware(authClient, db)

	// AuthHandlerの初期化とルートの登録
	authHandler := handler.NewAuthHandler(db, authClient)

	RegisterAuthRoutes(e, authClient, authHandler, authMiddleware)
}

func RegisterAuthRoutes(e *echo.Echo, authClient *auth.Client, authHandler *handler.AuthHandler, authMiddleware *middleware.AuthMiddleware) {
	// 認証不要のエンドポイント
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// 認証関連のエンドポイント（認証不要）
	auth := e.Group("/auth")
	auth.POST("/signin", authHandler.SignIn)        // サインイン（新規登録も含む）
	auth.POST("/validate", authHandler.ValidateToken)
	auth.POST("/signout", authHandler.SignOut)      // サインアウト

	// 認証が必要なエンドポイント
	api := e.Group("/api")
	api.Use(authMiddleware.FirebaseAuthMiddleware())

	// ユーザー関連のエンドポイント
	users := api.Group("/users")
	users.GET("/me", authHandler.GetCurrentUser)      // ユーザー情報取得
	users.PUT("/me", authHandler.UpdateUser)          // ユーザー情報更新
	users.DELETE("/me", authHandler.DeleteUser)       // アカウント削除
	users.GET("/me/points", authHandler.GetUserPoints) // ポイント情報取得
}
