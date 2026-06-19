package main

import (
	"IM/http"
	"IM/mysql"
	"IM/tcp"
	"log"
	"time"
)

func main() {
	config := MustLoadConfig(".")
	log.Println("config loaded:", config)

	mysql.ConfigInit(config.DataSource)
	mysql.InitMessageConn(config.DataSource)

	server := tcp.NewServer(config.TCPAddr, config.TcpPort, 10*time.Second)
	server.AddHandler(tcp.Verify)
	server.AddHandler(tcp.Router)
	server.AddHandler(tcp.Echo)

	go server.Start()
	go func() {
		http.NewServer(config.HttpAddress, config.HttpPort).Start()
	}()

	tcp.NotifyServer(server)
}