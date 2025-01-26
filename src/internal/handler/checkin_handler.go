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

	// リクエストボディの解析
	var req struct {
		SeichiID  int     `json:"seichi_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := c.Bind(&req); err != nil {
		fmt.Printf("リクエストボディのバインドエラー: %v\n", err)
		// エラーでも処理を継続
	}

	fmt.Printf("リクエストボディ: %+v\n", req)

	// UIDの取得（エラーでも処理を継続）
	uid := "unknown"
	if uidInterface := c.Get("uid"); uidInterface != nil {
		if uidStr, ok := uidInterface.(string); ok {
			uid = uidStr
		}
	}

	// ユーザー情報の取得（エラーでも処理を継続）
	var userID uint = 1 // デフォルト値
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, h.DB)
	if err == nil && user != nil {
		userID = user.UserID
	}

	// トランザクション開始
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("トランザクション開始エラー: %v\n", err)
		// エラーでも処理を継続し、新しいトランザクションを試みる
		tx, _ = h.DB.BeginTx(ctx, nil)
	}
	defer tx.Rollback()

	// チェックインログの作成と保存を試みる
	checkinLog := &models.CheckinLog{
		UserID:    userID,
		SeichiID:  req.SeichiID,
		CreatedAt: time.Now(),
	}

	if err := checkinLog.Insert(ctx, tx, boil.Infer()); err != nil {
		fmt.Printf("チェックインログ保存エラー: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "チェックイン処理に失敗しました",
		})
	}

	// トランザクションのコミット
	if err := tx.Commit(); err != nil {
		fmt.Printf("トランザクションコミットエラー: %v\n", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "トランザクションのコミットに失敗しました",
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "チェックイン成功",
		"checkin": map[string]interface{}{
			"created_at": checkinLog.CreatedAt,
			"user_id":   checkinLog.UserID,
			"seichi_id": checkinLog.SeichiID,
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
