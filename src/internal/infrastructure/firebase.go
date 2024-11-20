package firebase

import (
	"context"
	"fmt"
	"os"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func InitializeFirebaseApp() (*firebase.App, error) {
	ctx := context.Background()

	// 環境変数から認証情報ファイルのパスを取得
	credPath := os.Getenv("FIREBASE_SDK_PATH")
	if credPath == "" {
		return nil, fmt.Errorf("FIREBASE_SDK_PATHの環境変数が設定されていません")
	}
	opt := option.WithCredentialsFile(credPath)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("firebase initialization error: %v", err)
	}

	return app, nil
}
