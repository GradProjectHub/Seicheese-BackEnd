package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"fmt"
)

func RegisterCheckinRoutes(e *echo.Echo, checkinHandler *handler.CheckinHandler, authMiddleware *middleware.AuthMiddleware) {
	fmt.Printf("チェックインルーター登録開始\n")

	// チェックイン関連のルーティンググループ
	checkinGroup := e.Group("/checkins")

	// デバッグログ用のミドルウェア
	checkinGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Printf("===== チェックインリクエスト受信 =====\n")
			fmt.Printf("Method: %s\n", c.Request().Method)
			fmt.Printf("Path: %s\n", c.Request().URL.Path)
			fmt.Printf("Full URL: %s\n", c.Request().URL.String())
			return next(c)
		}
	})

	// チェックインの実行（認証なしでも実行可能に）
	checkinGroup.POST("", checkinHandler.Checkin)
	fmt.Printf("POSTルート登録: /checkins\n")

	// 認証が必要なエンドポイント
	authGroup := checkinGroup.Group("")
	authGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// チェックイン履歴の取得
	authGroup.GET("", checkinHandler.GetUserCheckins)
	fmt.Printf("GETルート登録: /checkins\n")

	fmt.Printf("チェックインルーター登録完了\n")
}
