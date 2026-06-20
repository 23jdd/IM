package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	chat := NewChatService()
	auth := NewAuthService()

	app := application.New(application.Options{
		Name:        "WeChatIM",
		Description: "IM 桌面客户端",
		Services: []application.Service{
			application.NewService(chat),
			application.NewService(auth),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 让桥接服务可向前端推送事件。
	chat.SetApp(app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "WeChatIM",
		Width:            1000,
		Height:           680,
		MinWidth:         800,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(245, 245, 245),
		URL:              "/",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
