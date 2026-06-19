package main

import (
	"IM/http"
	"IM/tcp"
)

func main() {
	config := MustLoadConfig()
	go tcp.NewServer(config.TCPAddr, config.TcpPort).Start()
	http.NewServer(config.TCPAddr, config.TcpPort).Start()
}
