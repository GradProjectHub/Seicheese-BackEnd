// Seicheese-Backend/src/internal/middleware/router/content_router.go
package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterContentRoutes(e *echo.Echo, contentHandler *handler.ContentHandler, authClient *auth.Client) {
	contentGroup := e.Group("/api/contents")
	contentGroup.Use(middleware.FirebaseAuthMiddleware(authClient))

	contentGroup.GET("/search", contentHandler.SearchContents)
	contentGroup.POST("/register", contentHandler.RegisterContent)
}
