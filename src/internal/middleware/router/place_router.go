package router

import (
	"seicheese/internal/handler"
	"seicheese/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
)

func RegisterPlaceRoutes(e *echo.Echo, placeHandler *handler.PlaceHandler, authClient *auth.Client) {
	placeGroup := e.Group("/api/places")
	placeGroup.Use(middleware.FirebaseAuthMiddleware(authClient))

	placeGroup.GET("", placeHandler.GetPlace)
	placeGroup.POST("", placeHandler.RegisterPlace)
}
