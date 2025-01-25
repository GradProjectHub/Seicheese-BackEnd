// Seicheese-Backend/src/internal/middleware/router/content_router.go
package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterContentRoutes(e *echo.Echo, contentHandler *handler.ContentHandler, authClient *auth.Client) {
	// 認証ミドルウェアの初期化
	authMiddleware := middleware.NewAuthMiddleware(authClient, contentHandler.DB)

	// コンテンツ関連のルーティンググループ
	contentGroup := e.Group("/contents")

	// すべてのエンドポイントで認証が必要
	contentGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// コンテンツの取得
	contentGroup.GET("", contentHandler.GetContents)

	// コンテンツの登録
	contentGroup.POST("", contentHandler.RegisterContent)
}
