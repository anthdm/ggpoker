package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

type GameVariant uint8

func (gv GameVariant) String() string {
	switch gv {
	case TexasHoldem:
		return "TEXAS HOLDEM"
	case Other:
		return "other"
	default:
		return "unknown"
	}
}

const (
	TexasHoldem GameVariant = iota
	Other
)

type ServerConfig struct {
	Version     string
	ListenAddr  string
	GameVariant GameVariant
}

type Server struct {
	ServerConfig

	transport *TCPTransport
	peers     map[net.Addr]*Peer
	addPeer   chan *Peer
	delPeer   chan *Peer
	msgCh     chan *Message

	gameState *GameState
}

func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		ServerConfig: cfg,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer, 10),
		delPeer:      make(chan *Peer),
		msgCh:        make(chan *Message),
		gameState:    NewGameState(),
	}

	tr := NewTCPTransport(s.ListenAddr)
	s.transport = tr

	tr.AddPeer = s.addPeer
	tr.DelPeer = s.addPeer

	return s
}

func (s *Server) Start() {
	go s.loop()

	logrus.WithFields(logrus.Fields{
		"port":    s.ListenAddr,
		"variant": s.GameVariant,
	}).Info("started new game server")

	s.transport.ListenAndAccept()
}

func (s *Server) sendPeerList(p *Peer) error {
	peerList := MessagePeerList{
		Peers: make([]string, len(s.peers)),
	}

	it := 0
	for addr := range s.peers {
		peerList.Peers[it] = addr.String()
		it++
	}

	msg := NewMessage(s.ListenAddr, peerList)
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	return p.Send(buf.Bytes())
}

func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		GameVariant: s.GameVariant,
		Version:     s.Version,
		GameStatus:  s.gameState.gameStatus,
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(hs); err != nil {
		return err
	}

	return p.Send(buf.Bytes())
}

// TODO(@anthdm): Right now we have some redundent code in registering new peers to the game network.
// maybe construct a new peer and handshake protocol after registering a plain connection?
func (s *Server) Connect(addr string) error {
	fmt.Printf("dialing from %s to  %s\n", s.ListenAddr, addr)

	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return err
	}

	peer := &Peer{
		conn:     conn,
		outbound: true,
	}

	s.addPeer <- peer
	fmt.Printf("dial from %s to  %s successfull\n", s.ListenAddr, addr)

	return s.SendHandshake(peer)
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.delPeer:
			logrus.WithFields(logrus.Fields{
				"addr": peer.conn.RemoteAddr(),
			}).Info("new player disconnected")

			delete(s.peers, peer.conn.RemoteAddr())

			// If a new peer connects to the server we send our handshake message and wait
			// for his reply.
		case peer := <-s.addPeer:
			if err := s.handshake(peer); err != nil {
				logrus.Errorf("%s:handshake with incoming player failed: %s ", s.ListenAddr, err)
				peer.conn.Close()

				delete(s.peers, peer.conn.RemoteAddr())
				continue
			}

			// TODO: check max players and other game state logic.
			go peer.ReadLoop(s.msgCh)

			if !peer.outbound {
				if err := s.SendHandshake(peer); err != nil {
					logrus.Errorf("failed to send handshake with peer: %s", err)
					peer.conn.Close()

					delete(s.peers, peer.conn.RemoteAddr())
					continue
				}

				if err := s.sendPeerList(peer); err != nil {
					logrus.Errorf("peerlist error: %s", err)
					continue
				}
			}

			logrus.WithFields(logrus.Fields{
				"addr": peer.conn.RemoteAddr(),
			}).Info("handshake successfull: new player connected")

			s.peers[peer.conn.RemoteAddr()] = peer

		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				panic(err)
			}
		}
	}
}

func (s *Server) handshake(p *Peer) error {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return err
	}

	if s.GameVariant != hs.GameVariant {
		return fmt.Errorf("gamevariant does not match %s", hs.GameVariant)
	}
	if s.Version != hs.Version {
		return fmt.Errorf("invalid version %s", hs.Version)
	}

	logrus.WithFields(logrus.Fields{
		"peer":       p.conn.RemoteAddr(),
		"version":    hs.Version,
		"variant":    hs.GameVariant,
		"gameStatus": hs.GameStatus,
	}).Info("received handshake")

	return nil
}

func (s *Server) handleMessage(msg *Message) error {
	logrus.WithFields(logrus.Fields{
		"from": msg.From,
	}).Info("received message")

	switch v := msg.Payload.(type) {
	case MessagePeerList:
		return s.handlePeerList(v)
	}
	return nil
}

// TODO FIXME: (@anthdm) maybe goroutine??
func (s *Server) handlePeerList(l MessagePeerList) error {
	fmt.Printf("peerlist => %+v\n", l)
	for i := 0; i < len(l.Peers); i++ {
		if err := s.Connect(l.Peers[i]); err != nil {
			logrus.Errorf("failed to dial peer: ", err)
			continue
		}
	}

	return nil
}

func init() {
	gob.Register(MessagePeerList{})
}
