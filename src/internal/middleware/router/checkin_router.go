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
	fmt.Printf("チェックイングループ作成: %v\n", checkinGroup)

	// デバッグログ用のミドルウェアを追加
	checkinGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Printf("===== チェックインリクエスト受信 =====\n")
			fmt.Printf("Method: %s\n", c.Request().Method)
			fmt.Printf("Path: %s\n", c.Request().URL.Path)
			fmt.Printf("Full URL: %s\n", c.Request().URL.String())
			fmt.Printf("Headers: %+v\n", c.Request().Header)
			fmt.Printf("RemoteAddr: %s\n", c.Request().RemoteAddr)
			fmt.Printf("================================\n")
			return next(c)
		}
	})

	// すべてのエンドポイントで認証が必要
	checkinGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// チェックイン履歴の取得
	checkinGroup.GET("", checkinHandler.GetUserCheckins)
	fmt.Printf("GETルート登録: /checkins\n")

	// チェックインの実行
	checkinGroup.POST("", checkinHandler.Checkin)
	fmt.Printf("POSTルート登録: /checkins\n")

	fmt.Printf("チェックインルーター登録完了\n")
}
