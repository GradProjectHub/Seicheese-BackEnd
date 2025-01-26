package handler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"seicheese/models"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/qm"
)

type UserHandler struct {
	DB *sql.DB
}

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{
		DB: db,
	}
}

type UserResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *UserHandler) RegisterUser(c echo.Context) error {
	var req struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "不正なリクエスト形式です",
		})
	}

	uid := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ユーザーIDが必要です",
		})
	}

	log.Printf("ユーザー登録開始: firebase_id=%s", uid)

	// ユーザーの存在確認
	exists, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).Exists(c.Request().Context(), h.DB)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ユーザー情報の確認に失敗しました",
		})
	}
	if exists {
		log.Printf("ユーザーが既に存在します: firebase_id=%s", uid)
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "既に登録されているユーザーです",
		})
	}

	// トランザクションを開始
	tx, err := h.DB.BeginTx(c.Request().Context(), nil)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "トランザクションの開始に失敗しました",
		})
	}

	// トランザクションのロールバック処理
	var txErr error
	defer func() {
		if txErr != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("トランザクションのロールバックに失敗: %v", rbErr)
			} else {
				log.Printf("トランザクションをロールバックしました")
			}
		}
	}()

	log.Printf("トランザクション開始")

	// ユーザーを作成
	now := time.Now()
	user := &models.User{
		FirebaseID: uid,
		CreatedAt:  null.TimeFrom(now),
		UpdatedAt:  null.TimeFrom(now),
	}

	if err := user.Insert(c.Request().Context(), tx, boil.Infer()); err != nil {
		txErr = err
		log.Printf("Error inserting user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ユーザーの登録に失敗しました",
		})
	}

	log.Printf("ユーザーを作成しました: user_id=%d", user.UserID)

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		txErr = err
		log.Printf("Error committing transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "トランザクションのコミットに失敗しました",
		})
	}

	log.Printf("トランザクションをコミットしました")

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"user_id":    user.UserID,
		"created_at": user.CreatedAt.Time,
		"updated_at": user.UpdatedAt.Time,
	})
}

func (h *UserHandler) GetUser(c echo.Context) error {
	uid := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ユーザーIDが必要です",
		})
	}

	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(c.Request().Context(), h.DB)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "ユーザーが見つかりません",
		})
	}
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ユーザー情報の取得に失敗しました",
		})
	}

	// ポイント情報の取得
	point, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(c.Request().Context(), h.DB)
	if err != nil && err != sql.ErrNoRows {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ポイント情報の取得に失敗しました",
		})
	}

	currentPoints := 0
	if point != nil {
		currentPoints = point.CurrentPoint
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":    user.UserID,
		"points":     currentPoints,
		"created_at": user.CreatedAt,
	})
}

// ポイント情報を更新するメソッド
func (h *UserHandler) UpdateUserPoints(c echo.Context) error {
	uid := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "ユーザーIDが必要です",
		})
	}

	var req struct {
		Points int `json:"points"`
	}

	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "不正なリクエスト形式です",
		})
	}

	// トランザクションを開始
	tx, err := h.DB.BeginTx(c.Request().Context(), nil)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "トランザクションの開始に失敗しました",
		})
	}

	// トランザクションのロールバック処理
	var txErr error
	defer func() {
		if txErr != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("トランザクションのロールバックに失敗: %v", rbErr)
			} else {
				log.Printf("トランザクションをロールバックしました")
			}
		}
	}()

	// ユーザーを取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(c.Request().Context(), h.DB)
	if err != nil {
		txErr = err
		log.Printf("Error fetching user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ユーザー情報の取得に失敗しました",
		})
	}

	// ポイント情報を更新
	point, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(c.Request().Context(), h.DB)
	if err != nil {
		txErr = err
		log.Printf("Error fetching point record: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ポイント情報の取得に失敗しました",
		})
	}

	point.CurrentPoint += req.Points
	updatedPoint, err := point.Update(c.Request().Context(), tx, boil.Infer())
	if err != nil {
		txErr = err
		log.Printf("Error updating point record: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ポイントの更新に失敗しました",
		})
	}

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		txErr = err
		log.Printf("Error committing transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "トランザクションのコミットに失敗しました",
		})
	}

	log.Printf("ポイント情報を更新しました: user_id=%d, new_points=%d", user.UserID, updatedPoint)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id": user.UserID,
		"points":  updatedPoint,
	})
}

// CreateUser handles user creation
func (h *UserHandler) CreateUser(ctx context.Context, firebaseID string) (*models.User, error) {
	log.Printf("新規ユーザー作成開始: firebase_id=%s", firebaseID)

	// トランザクション開始
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("トランザクション開始エラー: %v", err)
		return nil, fmt.Errorf("トランザクション開始に失敗しました: %v", err)
	}
	defer tx.Rollback()

	// 新規ユーザーの作成
	now := time.Now()
	user := &models.User{
		FirebaseID: firebaseID,
		CreatedAt:  null.TimeFrom(now),
		UpdatedAt:  null.TimeFrom(now),
	}

	log.Printf("新規ユーザー作成試行: firebase_id=%s", firebaseID)

	if err := user.Insert(ctx, tx, boil.Infer()); err != nil {
		log.Printf("ユーザー作成エラー: %v", err)
		return nil, fmt.Errorf("ユーザーの登録に失敗しました: %v", err)
	}

	log.Printf("ユーザーを作成しました: user_id=%d, firebase_id=%s", user.UserID, user.FirebaseID)

	// 初期ポイントの作成（新規登録ボーナス1000ポイント）
	point := &models.Point{
		UserID:       user.UserID,
		CurrentPoint: 1000, // 新規登録ボーナス
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	log.Printf("初期ポイント作成試行: user_id=%d", user.UserID)

	if err := point.Insert(ctx, tx, boil.Infer()); err != nil {
		log.Printf("初期ポイント作成エラー: %v", err)
		return nil, fmt.Errorf("初期ポイントの作成に失敗しました: %v", err)
	}

	log.Printf("初期ポイントを作成しました: user_id=%d, points=%d", user.UserID, point.CurrentPoint)

	// トランザクションのコミット
	if err := tx.Commit(); err != nil {
		log.Printf("トランザクションコミットエラー: %v", err)
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %v", err)
	}

	return user, nil
}

// GetOrCreateUser handles getting or creating a user
func (h *UserHandler) GetOrCreateUser(ctx context.Context, firebaseID string) (*models.User, bool, error) {
	if h == nil {
		return nil, false, fmt.Errorf("UserHandler is nil")
	}

	log.Printf("ユーザー取得または作成開始: firebase_id=%s", firebaseID)

	// ユーザーの存在確認
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(firebaseID),
	).One(ctx, h.DB)

	if err == sql.ErrNoRows {
		// ユーザーが存在しない場合は新規作成
		user, err = h.CreateUser(ctx, firebaseID)
		if err != nil {
			return nil, false, err
		}
		return user, true, nil
	} else if err != nil {
		log.Printf("ユーザー情報の取得に失敗: %v", err)
		return nil, false, fmt.Errorf("ユーザー情報の取得に失敗しました: %v", err)
	}

	return user, false, nil
}

// ポイント取得API
func (h *UserHandler) GetUserPoints(c echo.Context) error {
	ctx := c.Request().Context()
	uid := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "認証情報が必要です",
		})
	}

	// トランザクション開始
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("トランザクション開始エラー: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "トランザクションの開始に失敗しました")
	}
	defer tx.Rollback()

	// ユーザー情報の取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(ctx, tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "ユーザーが見つかりません")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗しました")
	}

	// ポイント情報の取得
	point, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(ctx, tx)
	if err != nil {
		if err == sql.ErrNoRows {
			// ポイントレコードが存在しない場合は新規作成
			point = &models.Point{
				UserID: user.UserID,
				CurrentPoint: 0,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := point.Insert(ctx, tx, boil.Infer()); err != nil {
				log.Printf("ポイント情報作成エラー: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の作成に失敗しました")
			}
		} else {
			log.Printf("ポイント情報取得エラー: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の取得に失敗しました")
		}
	}

	// ポイントログの取得（直近の履歴）
	pointLogs, err := models.PointLogs(
		models.PointLogWhere.UserID.EQ(user.UserID),
		qm.OrderBy("created_at DESC"),
		qm.Limit(10),
	).All(ctx, tx)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("ポイントログ取得エラー: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "ポイントログの取得に失敗しました")
	}

	// トランザクションのコミット
	if err := tx.Commit(); err != nil {
		log.Printf("トランザクションコミットエラー: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "トランザクションのコミットに失敗しました")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_points": point.CurrentPoint,
		"point_logs": pointLogs,
		"updated_at": point.UpdatedAt,
	})
}

func (h *UserHandler) GetMe(c echo.Context) error {
	ctx := c.Request().Context()
	
	// ミドルウェアからユーザーIDを取得
	token := c.Get("token").(*auth.Token)
	
	// ユーザー情報の取得
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(token.UID),
	).One(ctx, h.DB)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, "ユーザーが見つかりません")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "ユーザー情報の取得に失敗しました")
	}
	
	// ポイント情報の取得
	point, err := models.Points(
		models.PointWhere.UserID.EQ(user.UserID),
	).One(ctx, h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "ポイント情報の取得に失敗しました")
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": user,
		"point": point,
	})
}
