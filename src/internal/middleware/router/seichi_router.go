package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterSeichiRoutes(e *echo.Echo, seichiHandler *handler.SeichiHandler, authMiddleware *middleware.AuthMiddleware) {
	// 聖地関連のルーティンググループ
	seichiGroup := e.Group("/seichies")

	// 認証不要のエンドポイント
	seichiGroup.GET("", seichiHandler.GetSeichies)
	seichiGroup.GET("/search", seichiHandler.SearchSeichies)
	seichiGroup.GET("/bounds", seichiHandler.GetSeichiesInBounds)
	seichiGroup.GET("/recent", seichiHandler.GetRecentSeichies)

	// 認証が必要なエンドポイント
	authGroup := seichiGroup.Group("")
	authGroup.Use(authMiddleware.FirebaseAuthMiddleware())
	authGroup.POST("", seichiHandler.RegisterSeichi)
}
