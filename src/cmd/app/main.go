package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"seicheese/internal/handler"
	firebase "seicheese/internal/infrastructure"
	"seicheese/internal/infrastructure/database"
	"seicheese/internal/middleware"
	router "seicheese/internal/middleware/router"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// ログを標準出力に設定
	log.SetOutput(os.Stdout)

	e := echo.New()

	// CORS設定
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: []string{
			"https://seicheese.jp",
			"https://www.seicheese.jp",
			"http://localhost:3000",
			"http://localhost:8080",
		},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
	}))

	// Firebaseの初期化
	firebaseApp, err := firebase.InitializeFirebaseApp()
	if err != nil {
		log.Fatalf("Firebase initialization error: %v", err)
	}

	authClient, err := firebaseApp.Auth(context.Background())
	if err != nil {
		log.Fatalf("Auth client initialization error: %v", err)
	}

	// データベース接続
	dbConfig := database.NewDBConfig()
	db, err := database.InitializeDB(dbConfig)
	if err != nil {
		log.Fatalf("Database initialization error: %v", err)
	}
	defer db.Close()

	// ハンドラーの初期化
	authHandler := handler.NewAuthHandler(db, authClient)
	userHandler := handler.NewUserHandler(db)
	genreHandler := &handler.GenreHandler{
		DB: db,
	}
	seichiHandler := &handler.SeichiHandler{
		DB: db,
	}
	contentHandler := &handler.ContentHandler{
		DB: db,
	}
	checkinHandler := handler.NewCheckinHandler(db)

	// 認証ミドルウェアの初期化
	authMiddleware := middleware.NewAuthMiddleware(authClient, db)

	// ヘルスチェック用のエンドポイント
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// ルーターの登録
	router.RegisterAuthRoutes(e, authClient, authHandler, authMiddleware)
	router.RegisterUserRoutes(e, userHandler, authMiddleware)
	router.RegisterGenreRoutes(e, genreHandler, authMiddleware)
	router.RegisterSeichiRoutes(e, seichiHandler, authMiddleware)
	router.RegisterContentRoutes(e, contentHandler, authMiddleware)
	router.RegisterCheckinRoutes(e, checkinHandler, authMiddleware)

	// サーバー起動
	port := os.Getenv("PORT")
	if port == "" {
		port = "1300"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
