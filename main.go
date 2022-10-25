package main

import (
	"time"

	"github.com/anthdm/ggpoker/p2p"
)

func makeServerAndStart(addr string) *p2p.Server {
	cfg := p2p.ServerConfig{
		Version:     "GGPOKER V0.1-alpha",
		ListenAddr:  addr,
		GameVariant: p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	time.Sleep(1 * time.Second)

	return server
}

func main() {
	playerA := makeServerAndStart(":3000")
	playerB := makeServerAndStart(":4000")
	playerC := makeServerAndStart(":5000")
	playerD := makeServerAndStart(":6000")
	playerE := makeServerAndStart(":7000")
	playerF := makeServerAndStart(":8000")

	time.Sleep(time.Second * 1)
	playerB.Connect(playerA.ListenAddr)
	time.Sleep(time.Second * 1)
	playerC.Connect(playerB.ListenAddr)
	time.Sleep(time.Second * 1)
	playerD.Connect(playerC.ListenAddr)
	time.Sleep(time.Second * 1)
	playerE.Connect(playerD.ListenAddr)
	time.Sleep(time.Second * 1)
	playerF.Connect(playerE.ListenAddr)

	select {}
}
