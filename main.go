package main

import (
	"net/http"
	"time"

	"github.com/anthdm/ggpoker/p2p"
)

func makeServerAndStart(addr, apiAddr string) *p2p.Node {
	cfg := p2p.ServerConfig{
		Version:       "GGPOKER V0.2-alpha",
		ListenAddr:    addr,
		APIListenAddr: apiAddr,
		GameVariant:   p2p.TexasHoldem,
	}
	server := p2p.NewNode(cfg)
	go server.Start()

	time.Sleep(time.Millisecond * 200)

	return server
}

func main() {
	node1 := makeServerAndStart(":3000", ":3001")
	node2 := makeServerAndStart(":4000", ":4001")
	node3 := makeServerAndStart(":5000", ":5001")
	node4 := makeServerAndStart(":6000", ":6001")

	node2.Connect(node1.ListenAddr)
	node3.Connect(node2.ListenAddr)
	node4.Connect(node3.ListenAddr)

	go func() {
		time.Sleep(2 * time.Second)
		http.Get("http://localhost:3001/takeseat")
		// time.Sleep(2 * time.Second)
		// http.Get("http://localhost:4001/takeseat")
		// time.Sleep(2 * time.Second)
		// http.Get("http://localhost:5001/takeseat")
	}()

	select {}
	return
	// playerB := makeServerAndStart(":4000", ":4001") // sb
	// playerC := makeServerAndStart(":5000", ":5001") // bb
	// playerD := makeServerAndStart(":7000", ":7001") // bb + 2

	// go func() {
	// 	time.Sleep(time.Second * 2)
	// 	http.Get("http://localhost:3001/ready")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:4001/ready")

	// 	time.Sleep(time.Second * 2)
	// 	http.Get("http://localhost:5001/ready")

	// 	time.Sleep(time.Second * 2)
	// 	http.Get("http://localhost:7001/ready")

	// 	// [3000:D, 4000:sb, 5000:bb, 7000]
	// 	// PREFLOP
	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:4001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:5001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:7001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:3001/fold")

	// 	// // FLOP
	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:4001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:5001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:7001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:3001/fold")

	// 	// // TURN
	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:4001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:5001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:7001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:3001/fold")

	// 	// // RIVER
	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:4001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:5001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:7001/fold")

	// 	// time.Sleep(time.Second * 2)
	// 	// http.Get("http://localhost:3001/fold")

	// }()

	// time.Sleep(time.Millisecond * 200)
	// playerB.Connect(playerA.ListenAddr)

	// time.Sleep(time.Millisecond)
	// playerC.Connect(playerB.ListenAddr)

	// time.Sleep(time.Millisecond * 200)
	// playerD.Connect(playerC.ListenAddr)

	// select {}
}
