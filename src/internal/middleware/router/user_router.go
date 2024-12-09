package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterUserRoutes(e *echo.Echo, userHandler *handler.UserHandler, authClient *auth.Client) {
	// ユーザー関連のルーティンググループ
	userGroup := e.Group("/api/users")

	// すべてのエンドポイントで認証が必要
	userGroup.Use(middleware.FirebaseAuthMiddleware(authClient))

	// ユーザー情報の取得
	userGroup.GET("/me", userHandler.GetUser)

	// ユーザー登録
	userGroup.POST("", userHandler.RegisterUser)
}
