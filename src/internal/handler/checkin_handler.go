package handler

import (
	"database/sql"
	"net/http"
	"time"

	"seicheese/models"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type CheckinHandler struct {
	DB *sql.DB
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
