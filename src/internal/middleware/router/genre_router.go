package router

import (
	"database/sql"
	"seicheese/internal/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func RegisterGenreRoutes(e *echo.Echo, db *sql.DB) {
	// ミドルウェアの設定
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// ハンドラの初期化
	genreHandler := &handler.GenreHandler{DB: db}

	// ルーティングの設定
	e.GET("/api/genres", genreHandler.GetGenres)
}
