package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterGenreRoutes(e *echo.Echo, genreHandler *handler.GenreHandler, authClient *auth.Client) {
	// ジャンル関連のルーティンググループ
	genreGroup := e.Group("/api/genres")
	genreGroup.Use(middleware.FirebaseAuthMiddleware(authClient))

	// ルーティングの設定
	genreGroup.GET("", genreHandler.GetGenres)
}
