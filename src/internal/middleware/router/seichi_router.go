package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterSeichiRoutes(e *echo.Echo, seichiHandler *handler.SeichiHandler, authClient *auth.Client) {
	seichiGroup := e.Group("/api/seichi")
	seichiGroup.Use(middleware.FirebaseAuthMiddleware(authClient))

	seichiGroup.POST("/register", seichiHandler.RegisterSeichi)
	seichiGroup.GET("/list", seichiHandler.GetSeichies)
	seichiGroup.GET("/search", seichiHandler.SearchSeichies)
}
