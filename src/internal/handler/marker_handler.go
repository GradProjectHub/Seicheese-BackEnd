package handler

import (
    "database/sql"
    "net/http"
    "path/filepath"
    "seicheese/models"
    "strconv"

    "github.com/labstack/echo/v4"
)

type MarkerHandler struct {
    DB *sql.DB
}

// GetMarkers returns all available markers for the user
func (h *MarkerHandler) GetMarkers(c echo.Context) error {
    ctx := c.Request().Context()
    uid := c.Get("uid").(string)

    // ユーザーのポイントを取得
    user, err := models.Users(
        models.UserWhere.FirebaseID.EQ(uid),
    ).One(ctx, h.DB)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗しました")
    }

    // 利用可能なマーカーを取得
    markers, err := models.GetAvailableMarkers(ctx, h.DB, user.Points)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "マーカー情報の取得に失敗しました")
    }

    return c.JSON(http.StatusOK, markers)
}

// GetMarkerImage returns the marker image
func (h *MarkerHandler) GetMarkerImage(c echo.Context) error {
    ctx := c.Request().Context()
    markerID := c.Param("id")

    // マーカー情報を取得
    id, err := strconv.Atoi(markerID)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "不正なマーカーIDです")
    }

    marker, err := models.Markers(
        models.MarkerWhere.ID.EQ(id),
    ).One(ctx, h.DB)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "マーカーが見つかりません")
    }

    // 画像ファイルのパスを構築
    imagePath := filepath.Join("static/markers", marker.ImagePath)
    return c.File(imagePath)
} 