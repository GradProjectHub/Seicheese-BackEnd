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
func (h *CheckinHandler) calculatePoints(c echo.Context, userID int, seichiID int) (int, int, error) {
	ctx := c.Request().Context()
	
	// 基本ポイント
	points := BasePoints

	// 最新のチェックインログを取得
	lastCheckin, err := models.CheckinLogs(
		qm.Where("user_id = ? AND seichi_id = ?", userID, seichiID),
		qm.OrderBy("created_at DESC"),
		qm.Limit(1),
	).One(ctx, h.DB)

	if err != nil && err != sql.ErrNoRows {
		return 0, 0, err
	}

	// 初回チェックインボーナス
	if err == sql.ErrNoRows {
		points += FirstVisitBonus
	}

	// ポイントログの作成
	pointLog := &models.PointLog{
		UserID: userID,
		Point: points,
		Type: "checkin",
		CreatedAt: time.Now(),
	}

	// トランザクション開始
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	// ポイントログの保存
	if err := pointLog.Insert(ctx, tx, boil.Infer()); err != nil {
		return 0, 0, err
	}

	// ユーザーのポイントを更新
	userPoint, err := models.Points(
		models.PointWhere.UserID.EQ(userID),
	).One(ctx, tx)
	if err != nil {
		return 0, 0, err
	}

	userPoint.CurrentPoint += points
	if _, err := userPoint.Update(ctx, tx, boil.Infer()); err != nil {
		return 0, 0, err
	}

	// トランザクションのコミット
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	// スタンプIDの決定
	var stampID int
	if err == sql.ErrNoRows {
		stampID = FirstVisitStamp
	} else {
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
		"checkin": map[string]interface{}{
			"created_at": checkinLog.CreatedAt,
			"user_id": checkinLog.UserID,
			"seichi_id": checkinLog.SeichiID,
			"points_earned": points,
			"stamp_id": stampID,
		},
		"total_points": userPoint.CurrentPoint,
	})
}

// GetContentCheckins は作品ごとのチェックイン数を取得するハンドラー
func (h *CheckinHandler) GetContentCheckins(c echo.Context) error {
	ctx := c.Request().Context()
	uid := c.Get("uid").(string)

	// ユーザーの取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, h.DB)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "ユーザーが見つかりません",
		})
	}

	// 作品ごとのチェックイン数を取得
	query := `
		SELECT 
			c.content_id,
			c.content_name,
			COUNT(DISTINCT ch.checkin_id) as checkin_count
		FROM 
			contents c
			LEFT JOIN seichies s ON c.content_id = s.content_id
			LEFT JOIN checkins ch ON s.seichi_id = ch.seichi_id AND ch.user_id = $1
		GROUP BY 
			c.content_id, c.content_name
		ORDER BY 
			c.content_id
	`

	rows, err := h.DB.QueryContext(ctx, query, user.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "チェックイン数の取得に失敗しました",
		})
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var contentID int
		var contentName string
		var checkinCount int
		if err := rows.Scan(&contentID, &contentName, &checkinCount); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"content_id": contentID,
			"content_name": contentName,
			"checkin_count": checkinCount,
		})
	}

	return c.JSON(http.StatusOK, results)
}
