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

    // ユーザーのポイントを取得
    userPoint, err := models.Points(
        models.PointWhere.UserID.EQ(user.UserID),
    ).One(ctx, h.DB)
    if err != nil && err != sql.ErrNoRows {
        return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の取得に失敗")
    }

    currentPoints := 0
    if userPoint != nil {
        currentPoints = userPoint.CurrentPoint
    }

    // 利用可能なマーカーを取得
    markers, err := models.Markers(
        models.MarkersWhere.RequiredPoints.LTE(currentPoints),
        models.MarkersOrderBy.RequiredPoints.ASC(),
    ).All(ctx, h.DB)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "マーカー情報の取得に失敗")
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

    // マーカー画像の取得
    marker, err := models.Markers(
        models.MarkersWhere.ID.EQ(strconv.Itoa(id)),
    ).One(ctx, h.DB)
    if err != nil {
        if err == sql.ErrNoRows {
            return echo.NewHTTPError(http.StatusNotFound, "マーカーが見つかりません")
        }
        return echo.NewHTTPError(http.StatusInternalServerError, "マーカー情報の取得に失敗")
    }

    // 画像ファイルのパスを構築
    imagePath := filepath.Join("static/markers", marker.ImagePath)
    return c.File(imagePath)
} 