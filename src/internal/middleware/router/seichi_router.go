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

	// すべてのエンドポイントで認証が必要
	seichiGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// 聖地の取得
	seichiGroup.GET("", seichiHandler.GetSeichies)

	// 聖地の登録
	seichiGroup.POST("", seichiHandler.RegisterSeichi)

	seichiGroup.GET("/search", seichiHandler.SearchSeichies)
}
