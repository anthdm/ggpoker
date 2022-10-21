package main

import (
	"fmt"
	"time"

	"github.com/anthdm/ggpoker/p2p"
)

func main() {
	cfg := p2p.ServerConfig{
		Version:    "GGPOKER V0.1-alpha",
		ListenAddr: ":3000",
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	time.Sleep(1 * time.Second)

	remoteCfg := p2p.ServerConfig{
		Version:    "GGPOKER V0.1-alpha",
		ListenAddr: ":4000",
	}
	remoteServer := p2p.NewServer(remoteCfg)
	go remoteServer.Start()
	if err := remoteServer.Connect(":3000"); err != nil {
		fmt.Println(err)
	}

	select {}
}
