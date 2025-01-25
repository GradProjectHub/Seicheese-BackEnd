package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterPlaceRoutes(e *echo.Echo, placeHandler *handler.PlaceHandler, authClient *auth.Client) {
	// 認証ミドルウェアの初期化
	authMiddleware := middleware.NewAuthMiddleware(authClient, placeHandler.DB)

	// 場所関連のルーティンググループ
	placeGroup := e.Group("/places")

	// すべてのエンドポイントで認証が必要
	placeGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// 場所の取得
	placeGroup.GET("", placeHandler.GetPlace)

	// 場所の登録
	placeGroup.POST("", placeHandler.RegisterPlace)
}
