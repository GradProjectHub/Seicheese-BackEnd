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

func (h *CheckinHandler) GetUserCheckins(c echo.Context) error {
	ctx := c.Request().Context()
	uid := c.Get("uid").(string)

	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗")
	}

	checkins, err := models.CheckinLogs(
		qm.Where("user_id = ?", user.UserID),
		qm.OrderBy("created_at DESC"),
	).All(ctx, h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "チェックイン履歴の取得に失敗")
	}

	return c.JSON(http.StatusOK, checkins)
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
		return echo.NewHTTPError(http.StatusBadRequest, "不正なリクエスト形式")
	}

	// ユーザー情報の取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗")
	}

	// チェックインログの作成
	checkinLog := &models.CheckinLog{
		UserID:    user.UserID,
		SeichiID:  req.SeichiID,
		CreatedAt: time.Now(),
	}

	// データベースに保存
	if err := checkinLog.Insert(ctx, h.DB, boil.Infer()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "チェックイン処理に失敗")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "チェックイン成功",
		"checkin": checkinLog,
	})
}
