package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterCheckinRoutes(e *echo.Echo, checkinHandler *handler.CheckinHandler, authClient *auth.Client) {
	// 認証ミドルウェアの初期化
	authMiddleware := middleware.NewAuthMiddleware(authClient, checkinHandler.DB)

	// チェックイン関連のルーティンググループ
	checkinGroup := e.Group("/checkins")

	// すべてのエンドポイントで認証が必要
	checkinGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// チェックイン履歴の取得
	checkinGroup.GET("", checkinHandler.GetUserCheckins)

	// チェックインの実行
	checkinGroup.POST("", checkinHandler.Checkin)
}
