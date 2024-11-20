// Seicheese-Backend/src/internal/router/auth_router.go

package router

import (
	"net/http"
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterAuthRoutes(e *echo.Echo, authClient *auth.Client, authHandler *handler.AuthHandler) {
	// ヘルスチェックエンドポイントを追加（認証不要）
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// バックエンドでのトークンとバージョンの検証エンドポイントを追加
	e.POST("/auth/validate", authHandler.ValidateToken)

	authGroup := e.Group("")
	authGroup.Use(middleware.FirebaseAuthMiddleware(authClient))
	// ハンドラーの割り当て
	authGroup.POST("/auth/signin", authHandler.SignIn)
	authGroup.POST("/auth/signup", authHandler.SignUp)
}
