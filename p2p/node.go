package p2p

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/anthdm/ggpoker/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Node struct {
	ServerConfig

	gameState *GameState

	peerLock sync.RWMutex
	peers    map[string]proto.GossipClient

	broadcastch chan BroadcastTo

	proto.UnimplementedGossipServer
}

func NewNode(cfg ServerConfig) *Node {
	broadcastch := make(chan BroadcastTo, 1024)
	return &Node{
		ServerConfig: cfg,
		peers:        make(map[string]proto.GossipClient),
		broadcastch:  broadcastch,
		gameState:    NewGameState(cfg.ListenAddr, broadcastch),
	}
}

func (n *Node) Handshake(ctx context.Context, version *proto.Version) (*proto.Version, error) {
	client, err := makeGrpcClientConn(version.ListenAddr)
	if err != nil {
		return nil, err
	}

	n.addPeer(client, version)

	return n.getVersion(), nil
}

func (n *Node) addPeer(c proto.GossipClient, v *proto.Version) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	n.peers[v.ListenAddr] = c
	n.gameState.AddPlayer(v.ListenAddr)

	go func() {
		for _, addr := range v.PeerList {
			if err := n.Connect(addr); err != nil {
				fmt.Println("failed to connect: ", err)
				continue
			}
		}
	}()
}

func (n *Node) HandleTakeSeat(ctx context.Context, v *proto.TakeSeat) (*proto.Ack, error) {
	n.gameState.SetPlayerAtTable(v.Addr)
	return &proto.Ack{}, nil
}

func (n *Node) broadcast(bct BroadcastTo) {
	for _, addr := range bct.To {
		go func(addr string) {
			client, ok := n.peers[addr]
			if !ok {
				return
			}
			switch v := bct.Payload.(type) {
			case *proto.TakeSeat:
				_, err := client.HandleTakeSeat(context.TODO(), v)
				if err != nil {
					fmt.Printf("takeSeat broadcast error: %s\n", err)
				}

			case *proto.EncDeck:
				_, err := client.HandleEncDeck(context.TODO(), v)
				if err != nil {
					fmt.Printf("encDeck broadcast error: %s\n", err)
				}
			}
		}(addr)
	}
}

func (n *Node) loop() {
	for bt := range n.broadcastch {
		n.broadcast(bt)
	}
}

func (n *Node) getVersion() *proto.Version {
	return &proto.Version{
		Version:    "GGPOKER-0.0.1",
		ListenAddr: n.ListenAddr,
		PeerList:   n.getPeerList(),
	}
}

func (n *Node) getPeerList() []string {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()

	var (
		peers = make([]string, len(n.peers))
		i     = 0
	)
	for addr := range n.peers {
		peers[i] = addr
		i++
	}

	return peers
}

func (n *Node) canConnectWith(addr string) bool {
	if addr == n.ListenAddr {
		return false
	}
	for peerAddr := range n.peers {
		if peerAddr == addr {
			return false
		}
	}
	return true
}

func (n *Node) Connect(addr string) error {
	if !n.canConnectWith(addr) {
		return nil
	}
	client, err := makeGrpcClientConn(addr)
	if err != nil {
		return err
	}
	hs, err := client.Handshake(context.TODO(), n.getVersion())
	if err != nil {
		return err
	}
	n.addPeer(client, hs)
	return nil
}

func (n *Node) Start() error {
	grpcServer := grpc.NewServer()
	proto.RegisterGossipServer(grpcServer, n)

	go n.loop()

	ln, err := net.Listen("tcp", n.ListenAddr)
	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"port":       n.ListenAddr,
		"variant":    n.GameVariant,
		"maxPlayers": n.MaxPlayers,
	}).Info("started new poker game server")

	go func(n *Node) {
		apiServer := NewAPIServer(n.APIListenAddr, n.gameState)
		logrus.WithFields(logrus.Fields{
			"listenAddr": n.APIListenAddr,
		}).Info("starting API server")
		apiServer.Run()
	}(n)

	return grpcServer.Serve(ln)
}

func makeGrpcClientConn(addr string) (proto.GossipClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := proto.NewGossipClient(conn)
	return client, nil
}
