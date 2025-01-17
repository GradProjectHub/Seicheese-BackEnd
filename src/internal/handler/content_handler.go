// Seicheese-Backend/src/internal/handler/content_handler.go
package handler

import (
	"database/sql"
	"log"
	"net/http"
	"seicheese/models"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type ContentHandler struct {
	DB *sql.DB
}

// 作品登録API
func (h *ContentHandler) RegisterContent(c echo.Context) error {
	var req struct {
		Name    string `json:"content_name"`
		GenreID int    `json:"genre_id"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	log.Printf("Received content name: %s", req.Name)
	log.Printf("Received content genreId: %d", req.GenreID)

	// コンテンツ名の重複チェック
	exists, err := models.Contents(
		models.ContentWhere.ContentName.EQ(req.Name),
	).Exists(c.Request().Context(), h.DB)
	if err != nil {
		log.Printf("Error checking content existence: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "コンテンツの確認に失敗しました")
	}
	if exists {
		return echo.NewHTTPError(http.StatusConflict, "同じ名前のコンテンツが既に存在します")
	}

	content := models.Content{
		ContentName: req.Name,
		GenreID:     req.GenreID,
	}

	if err := content.Insert(c.Request().Context(), h.DB, boil.Infer()); err != nil {
		log.Printf("Error inserting content: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "作品の登録に失敗しました",
		})
	}

	log.Printf("Content registered successfully: %v", content)
	return c.JSON(http.StatusCreated, content)
}

// 検索API
func (h *ContentHandler) SearchContents(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusOK, []models.Content{}) // 空の配列を返す
	}

	contents, err := models.Contents(
		qm.Where("content_name LIKE ?", "%"+query+"%"),
	).All(c.Request().Context(), h.DB)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "検索に失敗しました",
		})
	}

	if len(contents) == 0 {
		return c.JSON(http.StatusOK, []models.Content{}) // 検索結果が0件の場合も空配列
	}

	return c.JSON(http.StatusOK, contents)
}
