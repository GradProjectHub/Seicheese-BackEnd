// Seicheese-Backend/src/internal/middleware/router/content_router.go
package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterContentRoutes(e *echo.Echo, contentHandler *handler.ContentHandler, authMiddleware *middleware.AuthMiddleware) {
	// コンテンツ関連のルーティンググループ
	contentGroup := e.Group("/contents")

	// 認証不要のエンドポイント
	contentGroup.GET("/search", contentHandler.SearchContents)
	contentGroup.GET("", contentHandler.GetContents)

	// 認証が必要なエンドポイント
	authGroup := contentGroup.Group("")
	authGroup.Use(authMiddleware.FirebaseAuthMiddleware())
	authGroup.POST("/register", contentHandler.RegisterContent)
}
