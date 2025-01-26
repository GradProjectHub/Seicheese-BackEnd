package handler

import (
	"database/sql"
	"fmt"
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
	var stampID int
	if err == sql.ErrNoRows {
		points += FirstVisitBonus
		stampID = FirstVisitStamp
	}

	// チェックイン回数を取得
	checkinCount, err := models.CheckinLogs(
		qm.Where("user_id = ? AND seichi_id = ?", userID, seichiID),
	).Count(ctx, h.DB)
	if err != nil {
		return 0, 0, err
	}

	// 5回訪問スタンプ
	if checkinCount == 4 { // 現在のチェックインで5回目
		stampID = FiveVisitsStamp
		points += ConsecutiveDaysBonus
	}

	// 10回訪問スタンプ
	if checkinCount == 9 { // 現在のチェックインで10回目
		stampID = TenVisitsStamp
		points += SpecialEventBonus
	}

	// 連続訪問ボーナスの確認
	if lastCheckin != nil {
		lastVisit := lastCheckin.CreatedAt
		if time.Since(lastVisit) <= 24*time.Hour {
			points += ConsecutiveDaysBonus
		}
	}

	return points, stampID, nil
}

func (h *CheckinHandler) Checkin(c echo.Context) error {
	ctx := c.Request().Context()

	// リクエストの詳細をログ出力
	fmt.Printf("チェックインリクエスト - Method: %s, Path: %s\n", c.Request().Method, c.Request().URL.Path)
	fmt.Printf("リクエストヘッダー: %+v\n", c.Request().Header)

	// UIDの取得
	uid := c.Get("uid").(string)
	if uid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"message": "ユーザーIDが必要です",
		})
	}

	// リクエストボディの解析
	var req struct {
		SeichiID  int     `json:"seichi_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := c.Bind(&req); err != nil {
		fmt.Printf("リクエストボディのバインドエラー: %v\n", err)
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"message": "不正なリクエスト形式",
		})
	}

	fmt.Printf("リクエストボディ: %+v\n", req)

	// 聖地の存在確認
	seichi, err := models.Seichies(
		models.SeichiWhere.SeichiID.EQ(req.SeichiID),
	).One(ctx, h.DB)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{
				"message": "指定された聖地が見つかりません",
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "聖地情報の取得に失敗しました",
		})
	}

	// ユーザー情報の取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, h.DB)
	if err != nil {
		fmt.Printf("ユーザー情報取得エラー: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "ユーザー情報の取得に失敗しました",
		})
	}

	// ポイントとスタンプの計算
	points, stampID, err := h.calculatePoints(c, user.UserID, req.SeichiID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "ポイント計算に失敗しました",
		})
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
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "トランザクション開始に失敗しました",
		})
	}
	defer tx.Rollback()

	// チェックインログの保存
	if err := checkinLog.Insert(ctx, tx, boil.Infer()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "チェックイン処理に失敗しました",
		})
	}

	// ユーザーのポイントを取得
	userPoint, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(ctx, tx)
	if err != nil && err != sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "ポイント情報の取得に失敗しました",
		})
	}

	// ポイントレコードが存在しない場合は新規作成
	if err == sql.ErrNoRows {
		userPoint = &models.Point{
			UserID: user.UserID,
			CurrentPoint: points,
		}
		if err := userPoint.Insert(ctx, tx, boil.Infer()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
				"message": "ポイント情報の作成に失敗しました",
			})
		}
	} else {
		// 既存のポイントを更新
		userPoint.CurrentPoint += points
		if _, err := userPoint.Update(ctx, tx, boil.Infer()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
				"message": "ポイント更新に失敗しました",
			})
		}
	}

	// トランザクションのコミット
	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "トランザクションのコミットに失敗しました",
		})
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
