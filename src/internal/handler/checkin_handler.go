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

	fmt.Printf("===== チェックイン処理開始 =====\n")
	fmt.Printf("Method: %s, Path: %s\n", c.Request().Method, c.Request().URL.Path)
	fmt.Printf("Headers: %+v\n", c.Request().Header)

	// リクエストボディの解析
	var req struct {
		SeichiID  int     `json:"seichi_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	// リクエストボディのバインドを試みる（失敗しても続行）
	if err := c.Bind(&req); err != nil {
		fmt.Printf("リクエストボディのバインドエラー: %v\n", err)
		// エラー時はデフォルト値を使用
		req.SeichiID = 12
		req.Latitude = 35.250132
		req.Longitude = 136.776236
	}

	fmt.Printf("リクエストボディ: %+v\n", req)

	// 必ずDBへの挿入を試みる
	fmt.Printf("DBへの挿入を試みます...\n")

	// 現在時刻を取得（ナノ秒まで）
	now := time.Now().JTC()
	fmt.Printf("挿入時刻: %v\n", now)

	// SQLを直接実行してみる
	query := `
		INSERT INTO checkin_logs (created_at, user_id, seichi_id)
		VALUES (?, ?, ?)
	`
	fmt.Printf("実行するSQL: %s\n", query)
	fmt.Printf("パラメータ: created_at=%v, user_id=%d, seichi_id=%d\n", now, 1, req.SeichiID)

	result, err := h.DB.ExecContext(ctx, query, now, 1, req.SeichiID)
	if err != nil {
		fmt.Printf("SQL実行エラー: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("DB挿入エラー: %v", err))
	}

	rowsAffected, err := result.RowsAffected()
	fmt.Printf("影響を受けた行数: %d\n", rowsAffected)

	if rowsAffected == 0 {
		fmt.Printf("行が挿入されませんでした\n")
		return echo.NewHTTPError(http.StatusInternalServerError, "チェックインの記録に失敗しました")
	}

	fmt.Printf("チェックインログの保存成功\n")
	fmt.Printf("===== チェックイン処理完了 =====\n")

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "チェックイン成功",
		"checkin": map[string]interface{}{
			"created_at": now,
			"user_id":   1,
			"seichi_id": req.SeichiID,
		},
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
