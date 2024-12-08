package handler

import (
	"database/sql"
	"net/http"
	"time"

	"seicheese/models"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type CheckinHandler struct {
	DB *sql.DB
}

type CheckinResponse struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	SeichiID   int       `json:"seichi_id"`
	CreatedAt  time.Time `json:"created_at"`
	SeichiName string    `json:"seichi_name,omitempty"`
}

func (h *CheckinHandler) Checkin(c echo.Context) error {
	ctx := c.Request().Context()

	// UIDの取得
	uid := c.Get("uid").(string)
	if uid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ユーザーIDが必要です")
	}

	// リクエストボディの解析
	var req struct {
		SeichiID int `json:"seichi_id"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "不正なリクエスト形式です")
	}

	// 聖地の存在確認
	seichi, err := models.Seichies(
		qm.Where("seichi_id = ?", req.SeichiID),
	).One(ctx, h.DB)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "指定された聖地が見つかりません")
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "聖地の確認に失敗しました")
	}

	// ユーザー情報の取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗しました")
	}

	// 24時間以内の重複チェックイン防止
	exists, err := models.CheckinLogs(
		qm.Where("user_id = ? AND seichi_id = ? AND created_at > ?",
			user.UserID, req.SeichiID, time.Now().Add(-24*time.Hour)),
	).Exists(ctx, h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "チェックイン履歴の確認に失敗しました")
	}
	if exists {
		return echo.NewHTTPError(http.StatusConflict, "24時間以内に同じ聖地へのチェックインがあります")
	}

	checkinLog := &models.CheckinLog{
		UserID:    user.UserID,
		SeichiID:  req.SeichiID,
		CreatedAt: time.Now(),
	}

	if err := checkinLog.Insert(ctx, h.DB, boil.Infer()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "チェックイン処理に失敗しました")
	}

	return c.JSON(http.StatusOK, CheckinResponse{
		ID:         checkinLog.CheckinID,
		UserID:     checkinLog.UserID,
		SeichiID:   checkinLog.SeichiID,
		CreatedAt:  checkinLog.CreatedAt,
		SeichiName: seichi.SeichiName,
	})
}
