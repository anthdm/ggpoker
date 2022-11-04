package p2p

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type PlayersReady struct {
	mu           sync.RWMutex
	recvStatutes map[string]bool
}

func NewPlayersReady() *PlayersReady {
	return &PlayersReady{
		recvStatutes: make(map[string]bool),
	}
}

func (pr *PlayersReady) haveRecv(from string) bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	_, ok := pr.recvStatutes[from]

	return ok
}

func (pr *PlayersReady) addRecvStatus(from string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recvStatutes[from] = true
}

func (pr *PlayersReady) len() int {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return len(pr.recvStatutes)
}

func (pr *PlayersReady) clear() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recvStatutes = make(map[string]bool)
}

type Game struct {
	listenAddr  string
	broadcastch chan BroadcastTo

	// currentStatus should be atomically accessable.
	currentStatus GameStatus
	// currentDealer should be atomically accessable.
	// NOTE: this will be -1 when the game is in a bootstrapped state.
	currentDealer int32

	playersReady *PlayersReady

	playersList PlayersList
}

func NewGame(addr string, bc chan BroadcastTo) *Game {
	g := &Game{
		listenAddr:    addr,
		broadcastch:   bc,
		currentStatus: GameStatusConnected,
		playersReady:  NewPlayersReady(),
		playersList:   PlayersList{},
		currentDealer: -1,
	}

	g.playersList = append(g.playersList, addr)

	go g.loop()

	return g
}

func (g *Game) setStatus(s GameStatus) {
	// Only update the status when the status is different.
	if g.currentStatus != s {
		atomic.StoreInt32((*int32)(&g.currentStatus), (int32)(s))
	}
}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealer := g.playersList[0] // currentDealer == -1
	if g.currentDealer > -1 {
		currentDealer = g.playersList[g.currentDealer]
	}
	return currentDealer, g.listenAddr == currentDealer
}

func (g *Game) SetPlayerReady(from string) {
	logrus.WithFields(logrus.Fields{
		"we":     g.listenAddr,
		"player": from,
	}).Info("setting player status to ready")

	g.playersReady.addRecvStatus(from)

	// If we don't have enough players the round cannot be started.
	if g.playersReady.len() < 2 {
		return
	}

	// In the case we have enough players. hence, the round can be started.
	// FIXME:(@anthdm)
	// g.playersReady.clear()

	// we need to check if we are the dealer of the current round.
	if _, ok := g.getCurrentDealerAddr(); ok {
		g.InitiateShuffleAndDeal()
	}
}

func (g *Game) ShuffleAndEncrypt(from string, deck [][]byte) error {
	prevPlayerAddr := g.playersList[g.getPrevPositionOnTable()]
	if from != prevPlayerAddr {
		return fmt.Errorf("received encrypted deck from the wrong player (%s) should be (%s)", from, prevPlayerAddr)
	}

	_, isDealer := g.getCurrentDealerAddr()
	if isDealer && from == prevPlayerAddr {
		logrus.Info("shuffle round complete")
		return nil
	}

	dealToPlayer := g.playersList[g.getNextPositionOnTable()]

	logrus.WithFields(logrus.Fields{
		"recvFromPlayer":  from,
		"we":              g.listenAddr,
		"dealingToPlayer": dealToPlayer,
	}).Info("received cards and going to shuffle")

	// TODO:(@anthdm) encryption and shuffle
	// TODO: get this player out of a deterministic (sorted) list.

	g.sendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayer)
	g.setStatus(GameStatusDealing)

	return nil
}

func (g *Game) InitiateShuffleAndDeal() {
	dealToPlayerAddr := g.playersList[g.getNextPositionOnTable()]
	g.setStatus(GameStatusDealing)
	g.sendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayerAddr)

	logrus.WithFields(logrus.Fields{
		"we": g.listenAddr,
		"to": dealToPlayerAddr,
	}).Info("dealing cards")
}

func (g *Game) SetReady() {
	g.playersReady.addRecvStatus(g.listenAddr)
	g.sendToPlayers(MessageReady{}, g.getOtherPlayers()...)
	g.setStatus(GameStatusPlayerReady)
}

func (g *Game) sendToPlayers(payload any, addr ...string) {
	g.broadcastch <- BroadcastTo{
		To:      addr,
		Payload: payload,
	}

	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"player":  addr,
		"we":      g.listenAddr,
	}).Info("sending payload to player")
}

func (g *Game) AddPlayer(from string) {
	// If the player is being added to the game. We are going to assume
	// that he is ready to play.
	g.playersList = append(g.playersList, from)
	sort.Sort(g.playersList)
}

func (g *Game) loop() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		<-ticker.C

		currentDealerAddr, _ := g.getCurrentDealerAddr()
		logrus.WithFields(logrus.Fields{
			"we":            g.listenAddr,
			"players":       g.playersList,
			"status":        g.currentStatus,
			"currentDealer": currentDealerAddr,
			// "playersReady": g.playersReady.recvStatutes,
		}).Info()
	}
}

func (g *Game) getOtherPlayers() []string {
	players := []string{}

	for _, addr := range g.playersList {
		if addr == g.listenAddr {
			continue
		}
		players = append(players, addr)
	}

	return players
}

// getPositionOnTable return the index of our own position on the table.
func (g *Game) getPositionOnTable() int {
	for i := 0; i < len(g.playersList); i++ {
		if g.playersList[i] == g.listenAddr {
			return i
		}
	}

	panic("player does not exist in the playersList; that should not happen!!!")
}

func (g *Game) getPrevPositionOnTable() int {
	ourPosition := g.getPositionOnTable()

	// if we are the in the first position on the table we need to return the last
	// index of the PlayersList.
	if ourPosition == 0 {
		return len(g.playersList) - 1
	}

	return ourPosition - 1
}

// getNextPositionOnTable returns the index of the next player in the PlayersList.
func (g *Game) getNextPositionOnTable() int {
	ourPosition := g.getPositionOnTable()

	// check if we are on the last position of the table, if so return first index 0.
	if ourPosition == len(g.playersList)-1 {
		return 0
	}

	return ourPosition + 1
}

func (g *Game) getNextReadyPlayer(pos int) string {
	nextPos := g.getNextPositionOnTable()
	nextPlayerAddr := g.playersList[nextPos]

	// fmt.Printf("%+v\n", )

	if g.playersReady.haveRecv(nextPlayerAddr) {
		return nextPlayerAddr
	}
	return g.getNextReadyPlayer(nextPos + 1)
}

// type GameState struct {
// 	listenAddr  string
// 	broadcastch chan BroadcastTo
// 	isDealer    bool // should be atomic accessable !

// 	gameStatus GameStatus // should be atomic accessable !

// 	playersList PlayersList
// 	playersLock sync.RWMutex
// 	players     map[string]*Player
// }

// func NewGameState(addr string, broadcastch chan BroadcastTo) *GameState {
// 	g := &GameState{
// 		listenAddr:  addr,
// 		broadcastch: broadcastch,
// 		isDealer:    false,
// 		gameStatus:  GameStatusWaitingForCards,
// 		players:     make(map[string]*Player),
// 	}

// 	g.AddPlayer(addr, GameStatusWaitingForCards)

// 	go g.loop()

// 	return g
// }

// // TODO:(@anthdm) Check other read and write occurences of the GameStatus!
// func (g *GameState) SetStatus(s GameStatus) {
// 	// Only update the status when the status is different.
// 	if g.gameStatus != s {
// 		atomic.StoreInt32((*int32)(&g.gameStatus), (int32)(s))
// 		g.SetPlayerStatus(g.listenAddr, s)
// 	}
// }

// func (g *GameState) playersWaitingForCards() int {
// 	totalPlayers := 0
// 	for i := 0; i < len(g.playersList); i++ {
// 		if g.playersList[i].Status == GameStatusWaitingForCards {
// 			totalPlayers++
// 		}
// 	}
// 	return totalPlayers
// }

// func (g *GameState) CheckNeedDealCards() {
// 	playersWaiting := g.playersWaitingForCards()

// 	if playersWaiting == len(g.players) &&
// 		g.isDealer &&
// 		g.gameStatus == GameStatusWaitingForCards {

// 		logrus.WithFields(logrus.Fields{
// 			"addr": g.listenAddr,
// 		}).Info("need to deal cards")

// 		g.InitiateShuffleAndDeal()
// 	}
// }

// func (g *GameState) GetPlayersWithStatus(s GameStatus) []string {
// 	players := []string{}
// 	for addr, player := range g.players {
// 		if player.Status == s {
// 			players = append(players, addr)
// 		}
// 	}
// 	return players
// }

// // InitiateShuffleAndDeal is only used for the "real" dealer. The actual "button player"
// func (g *GameState) InitiateShuffleAndDeal() { //
//dealToPlayer := g.playersList[g.getNextPositionOnTable()]

// 	g.SetStatus(GameStatusShuffleAndDeal)
// 	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
// }

// func (g *GameState) SendToPlayer(addr string, payload any) {
// 	g.broadcastch <- BroadcastTo{
// 		To:      []string{addr},
// 		Payload: payload,
// 	}

// 	logrus.WithFields(logrus.Fields{
// 		"payload": payload,
// 		"player":  addr,
// 	}).Info("sending payload to player")
// }

// func (g *GameState) SendToPlayersWithStatus(payload any, s GameStatus) {
// 	players := g.GetPlayersWithStatus(s)

// 	g.broadcastch <- BroadcastTo{
// 		To:      players,
// 		Payload: payload,
// 	}

// 	logrus.WithFields(logrus.Fields{
// 		"payload": payload,
// 		"players": players,
// 	}).Info("sending to players")
// }

// func (g *GameState) SetPlayerStatus(addr string, status GameStatus) {
// 	player, ok := g.players[addr]
// 	if !ok {
// 		panic("player could not be found, altough it should exist")
// 	}

// 	player.Status = status

// 	g.CheckNeedDealCards()
// }

// func (g *GameState) AddPlayer(addr string, status GameStatus) {
// 	g.playersLock.Lock()
// 	defer g.playersLock.Unlock()

// 	player := &Player{
// 		ListenAddr: addr,
// 	}
// 	g.players[addr] = player
// 	g.playersList = append(g.playersList, player)
// 	sort.Sort(g.playersList)

// 	// Set the player status also when we add the player!
// 	g.SetPlayerStatus(addr, status)

// 	logrus.WithFields(logrus.Fields{
// 		"addr":   addr,
// 		"status": status,
// 	}).Info("new player joined")
// }

// func (g *GameState) loop() {
// 	ticker := time.NewTicker(time.Second * 5)

// 	for {
// 		<-ticker.C
// 		logrus.WithFields(logrus.Fields{
// 			"we":      g.listenAddr,
// 			"players": g.playersList,
// 			"status":  g.gameStatus,
// 		}).Info()
// 	}
// }

type PlayersList []string

func (list PlayersList) Len() int { return len(list) }
func (list PlayersList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
func (list PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i][1:])
	portJ, _ := strconv.Atoi(list[j][1:])

	return portI < portJ
}

// type PlayersList []*Player

// func (list PlayersList) Len() int { return len(list) }
// func (list PlayersList) Swap(i, j int) {
// 	list[i], list[j] = list[j], list[i]
// }
// func (list PlayersList) Less(i, j int) bool {
// 	portI, _ := strconv.Atoi(list[i].ListenAddr[1:])
// 	portJ, _ := strconv.Atoi(list[j].ListenAddr[1:])

// 	return portI < portJ
// }

// type Player struct {
// 	Status     GameStatus
// 	ListenAddr string // [":3000", ":5000", ":4000]"
// }

// func (p *Player) String() string {
// 	return fmt.Sprintf("%s:%s", p.ListenAddr, p.Status)
// }
