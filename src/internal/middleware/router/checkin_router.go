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

	// 直接ルートに登録
	e.POST("/checkins", checkinHandler.Checkin, authMiddleware.FirebaseAuthMiddleware())
	fmt.Printf("POSTルート登録: /checkins\n")

	// GETルートも同様に直接登録
	e.GET("/checkins", checkinHandler.GetUserCheckins, authMiddleware.FirebaseAuthMiddleware())
	fmt.Printf("GETルート登録: /checkins\n")

	fmt.Printf("チェックインルーター登録完了\n")
}
