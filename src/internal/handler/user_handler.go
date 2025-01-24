package handler

import (
	"database/sql"
	"log"
	"net/http"
	"seicheese/models"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/null"
)

type UserHandler struct {
	DB *sql.DB
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
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "既に登録されているユーザーです",
		})
	}

	user := &models.User{
		FirebaseID: uid,
		CreatedAt:  null.TimeFrom(time.Now()),
		UpdatedAt:  null.TimeFrom(time.Now()),
	}

	tx, err := h.DB.BeginTx(c.Request().Context(), nil)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "トランザクションの開始に失敗しました",
		})
	}
	defer tx.Rollback()

	if err := user.Insert(c.Request().Context(), tx, boil.Infer()); err != nil {
		log.Printf("Error inserting user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ユーザーの登録に失敗しました",
		})
	}

	// ポイントレコードを作成
	now := time.Now()
	newPoint := &models.Point{
		UserID:       user.UserID,
		CurrentPoint: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := newPoint.Insert(c.Request().Context(), tx, boil.Infer()); err != nil {
		log.Printf("Error inserting point record: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ポイントの初期化に失敗しました",
		})
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "トランザクションのコミットに失敗しました",
		})
	}

	return c.JSON(http.StatusCreated, UserResponse{
		ID:        user.UserID,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
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

	return c.JSON(http.StatusOK, UserResponse{
		ID:        user.UserID,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
	})
}
