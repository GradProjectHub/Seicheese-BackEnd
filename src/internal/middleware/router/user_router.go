package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"github.com/labstack/echo/v4"
)

func RegisterUserRoutes(e *echo.Echo, userHandler *handler.UserHandler, authMiddleware *middleware.AuthMiddleware) {
	// ユーザー関連のルーティンググループ
	userGroup := e.Group("/users")

	// 認証不要のエンドポイント
	userGroup.GET("/me/points", userHandler.GetUserPoints)

	// 認証が必要なエンドポイント
	authGroup := userGroup.Group("")
	authGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// ユーザー情報の取得
	authGroup.GET("/me", userHandler.GetUser)

	// ユーザー登録
	authGroup.POST("", userHandler.RegisterUser)
}
