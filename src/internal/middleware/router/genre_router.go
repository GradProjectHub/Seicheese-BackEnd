package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterGenreRoutes(e *echo.Echo, genreHandler *handler.GenreHandler, authMiddleware *middleware.AuthMiddleware) {
	// ジャンル関連のルーティンググループ
	genreGroup := e.Group("/genres")

	// 認証不要のエンドポイント
	genreGroup.GET("", genreHandler.GetGenres)

	// 認証が必要なエンドポイント（現時点では無し）
}
