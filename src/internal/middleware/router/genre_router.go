package router

import (
	"seicheese/internal/handler"

	"github.com/labstack/echo/v4"
)

func RegisterGenreRoutes(e *echo.Echo, genreHandler *handler.GenreHandler) {
	// ルーティングの設定
	e.GET("/api/genres", genreHandler.GetGenres)
}
