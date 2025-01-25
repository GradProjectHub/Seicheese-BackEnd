package router

import (
    "seicheese/internal/handler"
    "seicheese/internal/middleware"

    "firebase.google.com/go/v4/auth"
    "github.com/labstack/echo/v4"
)

func RegisterMarkerRoutes(e *echo.Echo, markerHandler *handler.MarkerHandler, authClient *auth.Client) {
    // 認証ミドルウェアの初期化
    authMiddleware := middleware.NewAuthMiddleware(authClient, markerHandler.DB)

    // マーカー関連のルーティンググループ
    markerGroup := e.Group("/markers")

    // すべてのエンドポイントで認証が必要
    markerGroup.Use(authMiddleware.FirebaseAuthMiddleware())

    // マーカーの取得
    markerGroup.GET("", markerHandler.GetMarkers)

    // マーカー画像の取得
    markerGroup.GET("/:id/image", markerHandler.GetMarkerImage)
} 