package main

import (
	"IM/http"
	"IM/tcp"
	"log"
)

func main() {
	config := MustLoadConfig(".")
	log.Println(config)
	go tcp.NewServer(config.TCPAddr, config.TcpPort).Start()
	http.NewServer(config.HttpAddress, config.HttpPort).Start()
}
