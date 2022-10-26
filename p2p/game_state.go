package p2p

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthdm/ggpoker/deck"
	"github.com/sirupsen/logrus"
)

type GameStatus int32

func (g GameStatus) String() string {
	switch g {
	case GameStatusWaitingForCards:
		return "WAITING FOR CARDS"
	case GameStatusReceivingCards:
		return "RECEIVING CARDS"
	case GameStatusDealing:
		return "DEALING"
	case GameStatusPreFlop:
		return "PRE FLP"
	case GameStatusFlop:
		return "FLOP"
	case GameStatusTurn:
		return "TURN"
	case GameStatusRiver:
		return "RIVER"
	default:
		return "unknown"
	}
}

const (
	GameStatusWaitingForCards GameStatus = iota
	GameStatusReceivingCards
	GameStatusDealing
	GameStatusPreFlop
	GameStatusFlop
	GameStatusTurn
	GameStatusRiver
)

type Player struct {
	Status GameStatus
}

type GameState struct {
	listenAddr  string
	broadcastch chan any
	isDealer    bool // should be atomic accessable !

	gameStatus GameStatus // should be atomic accessable !

	playersWaitingForCards int32
	playersLock            sync.RWMutex
	players                map[string]*Player
}

func NewGameState(addr string, broadcastch chan any) *GameState {
	g := &GameState{
		listenAddr:  addr,
		broadcastch: broadcastch,
		isDealer:    false,
		gameStatus:  GameStatusWaitingForCards,
		players:     make(map[string]*Player),
	}

	go g.loop()

	return g
}

// TODO:(@anthdm) Check other read and write occurences of the GameStatus!
func (g *GameState) SetStatus(s GameStatus) {
	atomic.StoreInt32((*int32)(&g.gameStatus), (int32)(s))
	// TODO make this an concurent safe operation preferably an atomic one!
	g.gameStatus = s
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

		g.DealCards()
	}
}

func (g *GameState) DealCards() {
	g.broadcastch <- MessageCards{Deck: deck.New()}
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

	g.players[addr] = new(Player)

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
				"players connected": g.LenPlayersConnectedWithLock(),
				"status":            g.gameStatus,
			}).Info()

		default:
		}
	}
}
