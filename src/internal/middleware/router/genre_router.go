package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterGenreRoutes(e *echo.Echo, genreHandler *handler.GenreHandler, authClient *auth.Client) {
	// 認証ミドルウェアの初期化
	authMiddleware := middleware.NewAuthMiddleware(authClient, genreHandler.DB)

	// ジャンル関連のルーティンググループ
	genreGroup := e.Group("/genres")

	// すべてのエンドポイントで認証が必要
	genreGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// ジャンルの取得
	genreGroup.GET("", genreHandler.GetGenres)

	// ジャンルの登録
	genreGroup.POST("", genreHandler.RegisterGenre)
}
