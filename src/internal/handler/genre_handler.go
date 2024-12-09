package handler

import (
	"database/sql"
	"net/http"
	"seicheese/models"

	"github.com/labstack/echo/v4"
)

type GenreHandler struct {
	DB *sql.DB
}

// ジャンル一覧取得API
func (h *GenreHandler) GetGenres(c echo.Context) error {
	genres, err := models.Genres().All(c.Request().Context(), h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "ジャンルの取得に失敗しました")
	}

	c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
	return c.JSON(http.StatusOK, genres)
}
