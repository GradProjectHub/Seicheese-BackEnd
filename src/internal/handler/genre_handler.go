package handler

import (
	"database/sql"
	"log"
	"net/http"
	"seicheese/models"

	"github.com/labstack/echo/v4"
)

type GenreHandler struct {
	DB *sql.DB
}

// ジャンル一覧取得API
func (h *GenreHandler) GetGenres(c echo.Context) error {
	log.Printf("GetGenres called")
	
	// リクエストの詳細をログ出力
	log.Printf("Request Headers: %+v", c.Request().Header)
	log.Printf("Authorization Header: %s", c.Request().Header.Get("Authorization"))

	genres, err := models.Genres().All(c.Request().Context(), h.DB)
	if err != nil {
		log.Printf("Error fetching genres: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ジャンルの取得に失敗しました")
	}

	log.Printf("Found %d genres", len(genres))
	if len(genres) > 0 {
		log.Printf("First genre: %+v", genres[0])
	}

	c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
	return c.JSON(http.StatusOK, genres)
}
