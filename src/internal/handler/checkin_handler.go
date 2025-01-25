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

// ポイント計算用の定数
const (
	BasePoints = 100
	FirstVisitBonus = 500
	ConsecutiveDaysBonus = 200
	SpecialEventBonus = 1000
)

// スタンプ獲得条件
const (
	FirstVisitStamp = 1
	FiveVisitsStamp = 2
	TenVisitsStamp = 3
	SpecialEventStamp = 4
)

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

// ポイント計算ロジック
func (h *CheckinHandler) calculatePoints(ctx echo.Context, userID uint, seichiID int) (int, int, error) {
	// 基本ポイント
	points := BasePoints

	// 初回訪問ボーナスのチェック
	exists, err := models.CheckinLogs(
		qm.Where("user_id = ? AND seichi_id = ?", userID, seichiID),
	).Exists(ctx.Request().Context(), h.DB)
	if err != nil {
		return 0, 0, err
	}
	if !exists {
		points += FirstVisitBonus
		return points, FirstVisitStamp, nil
	}

	// 連続訪問ボーナスのチェック
	yesterday := time.Now().AddDate(0, 0, -1)
	hasConsecutiveVisit, err := models.CheckinLogs(
		qm.Where("user_id = ? AND created_at >= ?", userID, yesterday),
	).Exists(ctx.Request().Context(), h.DB)
	if err != nil {
		return 0, 0, err
	}
	if hasConsecutiveVisit {
		points += ConsecutiveDaysBonus
	}

	// 訪問回数に基づくスタンプの決定
	visits, err := models.CheckinLogs(
		qm.Where("user_id = ? AND seichi_id = ?", userID, seichiID),
	).Count(ctx.Request().Context(), h.DB)
	if err != nil {
		return 0, 0, err
	}

	var stampID int
	switch {
	case visits == 4:
		stampID = FiveVisitsStamp
	case visits == 9:
		stampID = TenVisitsStamp
	default:
		stampID = 0
	}

	return points, stampID, nil
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

	// ポイントとスタンプの計算
	points, stampID, err := h.calculatePoints(c, user.UserID, req.SeichiID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "ポイント計算に失敗")
	}

	// チェックインログの作成
	checkinLog := &models.CheckinLog{
		UserID:    user.UserID,
		SeichiID:  req.SeichiID,
		CreatedAt: time.Now(),
		Points:    points,
		StampID:   stampID,
	}

	// トランザクション開始
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "トランザクション開始に失敗")
	}
	defer tx.Rollback()

	// チェックインログの保存
	if err := checkinLog.Insert(ctx, tx, boil.Infer()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "チェックイン処理に失敗")
	}

	// ユーザーのポイントを取得
	userPoint, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(ctx, tx)
	if err != nil && err != sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の取得に失敗")
	}

	// ポイントレコードが存在しない場合は新規作成
	if err == sql.ErrNoRows {
		userPoint = &models.Point{
			UserID: user.UserID,
			CurrentPoint: points,
		}
		if err := userPoint.Insert(ctx, tx, boil.Infer()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の作成に失敗")
		}
	} else {
		// 既存のポイントを更新
		userPoint.CurrentPoint += points
		if _, err := userPoint.Update(ctx, tx, boil.Infer()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "ポイント更新に失敗")
		}
	}

	// トランザクションのコミット
	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "トランザクションのコミットに失敗")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "チェックイン成功",
		"checkin": checkinLog,
		"points_earned": points,
		"total_points": userPoint.CurrentPoint,
		"stamp_id": stampID,
	})
}
