package handler

import (
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"seicheese/models"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type MarkerHandler struct {
	DB *sql.DB
}

// GetMarkers ユーザーが利用可能なマーカー一覧を取得
func (h *MarkerHandler) GetMarkers(c echo.Context) error {
	uid := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ユーザーIDが必要です",
		})
	}

	// ユーザー情報の取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(c.Request().Context(), h.DB)
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ユーザー情報の取得に失敗しました",
		})
	}

	// ポイント情報の取得
	point, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(c.Request().Context(), h.DB)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error fetching points: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ポイント情報の取得に失敗しました",
		})
	}

	currentPoints := 0
	if point != nil {
		currentPoints = point.CurrentPoint
	}

	// 利用可能なマーカーの取得
	markers, err := models.Markers(
		qm.Where("required_points <= ?", currentPoints),
		qm.OrderBy("required_points ASC"),
	).All(c.Request().Context(), h.DB)
	if err != nil {
		log.Printf("Error fetching markers: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "マーカー情報の取得に失敗しました",
		})
	}

	return c.JSON(http.StatusOK, markers)
}

// GetMarkerImage マーカー画像の取得
func (h *MarkerHandler) GetMarkerImage(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "不正なマーカーIDです",
		})
	}

	// マーカー画像の取得
	marker, err := models.Markers(
		qm.Where("id = ?", id),
	).One(c.Request().Context(), h.DB)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "マーカーが見つかりません",
			})
		}
		log.Printf("Error fetching marker: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "マーカー情報の取得に失敗しました",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"image_path": marker.ImagePath,
	})
} 