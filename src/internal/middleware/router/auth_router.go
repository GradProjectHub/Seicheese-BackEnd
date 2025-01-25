// Seicheese-Backend/src/internal/router/auth_router.go

package router

import (
	"net/http"
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterAuthRoutes(e *echo.Echo, authClient *auth.Client, authHandler *handler.AuthHandler, authMiddleware *middleware.AuthMiddleware) {
	// 認証不要のエンドポイント
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// 認証関連のエンドポイント（認証不要）
	auth := e.Group("/auth")
	auth.POST("/signin", authHandler.SignIn)        // サインイン（新規登録も含む）
	auth.POST("/validate", authHandler.ValidateToken)
	auth.POST("/signout", authHandler.SignOut)      // サインアウト

	// 認証が必要なエンドポイント
	users := e.Group("/users")
	users.Use(authMiddleware.FirebaseAuthMiddleware())

	// ユーザー関連のエンドポイント
	users.GET("/me", authHandler.GetCurrentUser)      // ユーザー情報取得
	users.PUT("/me", authHandler.UpdateUser)          // ユーザー情報更新
	users.DELETE("/me", authHandler.DeleteUser)       // アカウント削除
	users.GET("/me/points", authHandler.GetUserPoints) // ポイント情報取得
}
