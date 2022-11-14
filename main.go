package main

import (
	"net/http"
	"time"

	"github.com/anthdm/ggpoker/p2p"
)

func makeServerAndStart(addr, apiAddr string) *p2p.Server {
	cfg := p2p.ServerConfig{
		Version:       "GGPOKER V0.2-alpha",
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
	playerA := makeServerAndStart(":3000", ":3001") // dealer
	playerB := makeServerAndStart(":4000", ":4001") // sb
	// playerC := makeServerAndStart(":5000", ":5001") // bb
	// playerD := makeServerAndStart(":7000", ":7001") // bb + 2

	go func() {
		time.Sleep(time.Second * 2)
		http.Get("http://localhost:3001/ready")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:4001/ready")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:5001/ready")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:7001/ready")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:5001/ready")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:7001/ready")

		time.Sleep(time.Second * 5)
		http.Get("http://localhost:4001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:5001/fold")

		// time.Sleep(time.Second * 2)
		// http.Get("http://localhost:7001/fold")

		time.Sleep(time.Second * 2)
		http.Get("http://localhost:3001/fold")

	}()

	time.Sleep(time.Millisecond * 200)
	playerB.Connect(playerA.ListenAddr)

	// time.Sleep(time.Millisecond)
	// playerC.Connect(playerB.ListenAddr)

	// time.Sleep(time.Millisecond * 200)
	// playerD.Connect(playerC.ListenAddr)

	select {}
}
