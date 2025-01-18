package router

import (
    "seicheese/internal/handler"
    "seicheese/internal/middleware"

    "firebase.google.com/go/v4/auth"
    "github.com/labstack/echo/v4"
)

func RegisterMarkerRoutes(e *echo.Echo, markerHandler *handler.MarkerHandler, authClient *auth.Client) {
    // マーカー関連のルーティンググループ
    markerGroup := e.Group("/api/markers")

    // 認証が必要なエンドポイント
    markerGroup.Use(middleware.FirebaseAuthMiddleware(authClient))
    markerGroup.GET("", markerHandler.GetMarkers)

    // 画像取得は認証不要
    e.GET("/api/markers/:id/image", markerHandler.GetMarkerImage)
} 