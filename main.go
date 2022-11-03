package main

import (
	"time"

	"github.com/anthdm/ggpoker/p2p"
)

func makeServerAndStart(addr, apiAddr string) *p2p.Server {
	cfg := p2p.ServerConfig{
		Version:       "GGPOKER V0.1-alpha",
		ListenAddr:    addr,
		APIListenAddr: apiAddr,
		GameVariant:   p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	time.Sleep(time.Millisecond * 200)

	return server
}

func main() {
	playerA := makeServerAndStart(":3000", ":3001")
	playerB := makeServerAndStart(":4000", ":4001")
	playerC := makeServerAndStart(":5000", ":5001")
	playerD := makeServerAndStart(":7000", ":7001")
	//playerE := makeServerAndStart(":8000")

	time.Sleep(time.Millisecond * 200)
	playerB.Connect(playerA.ListenAddr)
	time.Sleep(time.Millisecond)
	playerC.Connect(playerB.ListenAddr)
	time.Sleep(time.Millisecond * 200)
	playerD.Connect(playerC.ListenAddr)
	// time.Sleep(time.Millisecond * 200)
	// playerF.Connect(playerE.ListenAddr)

	select {}
}
