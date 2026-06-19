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
	log.Println(config)
	mysql.ConfigInit(config.DataSource)
	server := tcp.NewServer(config.TCPAddr, config.TcpPort, 10*time.Second)
	server.AddHandler(tcp.Echo)
	go server.Start()
	http.NewServer(config.HttpAddress, config.HttpPort).Start()
}
