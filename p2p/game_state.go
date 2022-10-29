package p2p

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type Player struct {
	Status     GameStatus
	ListenAddr string
}

type GameState struct {
	listenAddr  string
	broadcastch chan BroadcastTo
	isDealer    bool // should be atomic accessable !

	gameStatus GameStatus // should be atomic accessable !

	playersWaitingForCards int32
	playersLock            sync.RWMutex
	playersList            []*Player
	players                map[string]*Player

	decksReceivedLock sync.RWMutex
	decksReceived     map[string]bool
}

func NewGameState(addr string, broadcastch chan BroadcastTo) *GameState {
	g := &GameState{
		listenAddr:    addr,
		broadcastch:   broadcastch,
		isDealer:      false,
		gameStatus:    GameStatusWaitingForCards,
		players:       make(map[string]*Player),
		decksReceived: make(map[string]bool),
	}

	go g.loop()

	return g
}

// TODO:(@anthdm) Check other read and write occurences of the GameStatus!
func (g *GameState) SetStatus(s GameStatus) {
	// Only update the status when the status is different.
	if g.gameStatus != s {
		atomic.StoreInt32((*int32)(&g.gameStatus), (int32)(s))
	}
}

func (g *GameState) AddPlayerWaitingForCards() {
	atomic.AddInt32(&g.playersWaitingForCards, 1)
}

func (g *GameState) CheckNeedDealCards() {
	playersWaiting := atomic.LoadInt32(&g.playersWaitingForCards)

	if playersWaiting == int32(len(g.players)) &&
		g.isDealer &&
		g.gameStatus == GameStatusWaitingForCards {

		logrus.WithFields(logrus.Fields{
			"addr": g.listenAddr,
		}).Info("need to deal cards")

		g.InitiateShuffleAndDeal()
	}
}

func (g *GameState) GetPlayersWithStatus(s GameStatus) []string {
	players := []string{}
	for addr, player := range g.players {
		if player.Status == s {
			players = append(players, addr)
		}
	}
	return players
}

func (g *GameState) SetDecksReceived(from string) {
	g.decksReceivedLock.Lock()
	g.decksReceived[from] = true
	g.decksReceivedLock.Unlock()
}

func (g *GameState) ShuffleAndEncrypt(from string, deck [][]byte) error {
	dealToPlayer := g.playersList[0]

	logrus.WithFields(logrus.Fields{
		"recvFromPlayer":  from,
		"we":              g.listenAddr,
		"dealingToPlayer": dealToPlayer,
	}).Info("received cards and going to shuffle")

	// TODO:(@anthdm) encryption and shuffle
	// TODO: get this player out of a deterministic (sorted) list.

	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
	g.SetStatus(GameStatusShuffleAndDeal)

	return nil
}

// InitiateShuffleAndDeal is only used for the "real" dealer. The actual "button player"
func (g *GameState) InitiateShuffleAndDeal() {
	dealToPlayer := g.playersList[0]

	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
	g.SetStatus(GameStatusShuffleAndDeal)
}

func (g *GameState) SendToPlayer(addr string, payload any) {
	g.broadcastch <- BroadcastTo{
		To:      []string{addr},
		Payload: payload,
	}

	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"player":  addr,
	}).Info("sending payload to player")
}

func (g *GameState) SendToPlayersWithStatus(payload any, s GameStatus) {
	players := g.GetPlayersWithStatus(s)

	g.broadcastch <- BroadcastTo{
		To:      players,
		Payload: payload,
	}

	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"players": players,
	}).Info("sending to players")
}

func (g *GameState) DealCards() {
	// g.broadcastch <- MessageCards{Deck: deck.New()}
}

func (g *GameState) SetPlayerStatus(addr string, status GameStatus) {
	player, ok := g.players[addr]

	if !ok {
		panic("player could not be found, altough it should exist")
	}
	player.Status = status

	g.CheckNeedDealCards()
}

func (g *GameState) LenPlayersConnectedWithLock() int {
	g.playersLock.RLock()
	defer g.playersLock.RUnlock()

	return len(g.players)
}

func (g *GameState) AddPlayer(addr string, status GameStatus) {
	g.playersLock.Lock()
	defer g.playersLock.Unlock()

	if status == GameStatusWaitingForCards {
		g.AddPlayerWaitingForCards()
	}

	player := &Player{
		ListenAddr: addr,
	}
	g.players[addr] = player
	g.playersList = append(g.playersList, player)

	// Set the player status also when we add the player!
	g.SetPlayerStatus(addr, status)

	logrus.WithFields(logrus.Fields{
		"addr":   addr,
		"status": status,
	}).Info("new player joined")
}

func (g *GameState) loop() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-ticker.C:
			logrus.WithFields(logrus.Fields{
				"we":                g.listenAddr,
				"players connected": g.LenPlayersConnectedWithLock(),
				"status":            g.gameStatus,
				"decksReceived":     g.decksReceived,
			}).Info()

		default:
		}
	}
}
