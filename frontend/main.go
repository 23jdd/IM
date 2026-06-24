package main

import (
	"embed"
	"flag"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// assets 内嵌的前端静态资源（构建产物 frontend/dist）。
//go:embed all:frontend/dist
var assets embed.FS

// main 解析命令行参数、初始化各服务并启动 Wails 桌面应用。
func main() {
	host := flag.String("host", "124.223.72.39", "目标服务器 IP / 主机名")
	httpPort := flag.Int("http-port", 8080, "后端 HTTP 端口")
	tcpPort := flag.Int("tcp-port", 9000, "后端 TCP 端口")
	flag.Parse()

	chat := NewChatService()
	auth := NewAuthService()
	local := NewLocalStore()

	auth.SetBaseURL(fmt.Sprintf("http://%s:%d", *host, *httpPort))
	chat.SetDefaultAddr(fmt.Sprintf("%s:%d", *host, *tcpPort))

	app := application.New(application.Options{
		Name:        "WeChatIM",
		Description: "IM 桌面客户端",
		Services: []application.Service{
			application.NewService(chat),
			application.NewService(auth),
			application.NewService(local),
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
