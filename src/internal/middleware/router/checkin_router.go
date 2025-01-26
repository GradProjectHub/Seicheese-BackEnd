package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"fmt"
)

func RegisterCheckinRoutes(e *echo.Echo, checkinHandler *handler.CheckinHandler, authMiddleware *middleware.AuthMiddleware) {
	// チェックイン関連のルーティンググループ
	checkinGroup := e.Group("/checkins")

	// デバッグログ用のミドルウェアを追加
	checkinGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fmt.Printf("チェックインリクエスト受信 - Method: %s, Path: %s\n", c.Request().Method, c.Request().URL.Path)
			fmt.Printf("リクエストヘッダー: %+v\n", c.Request().Header)
			return next(c)
		}
	})

	// すべてのエンドポイントで認証が必要
	checkinGroup.Use(authMiddleware.FirebaseAuthMiddleware())

	// チェックイン履歴の取得
	checkinGroup.GET("", checkinHandler.GetUserCheckins)

	// チェックインの実行
	checkinGroup.POST("", checkinHandler.Checkin)
}
