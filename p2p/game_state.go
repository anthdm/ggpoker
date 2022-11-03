package p2p

import (
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

	playersReady *PlayersReady
	playersList  PlayersList
}

func NewGame(addr string, bc chan BroadcastTo) *Game {
	g := &Game{
		listenAddr:    addr,
		broadcastch:   bc,
		currentStatus: GameStatusConnected,
		playersReady:  NewPlayersReady(),
		playersList:   PlayersList{},
	}

	go g.loop()

	return g
}

func (g *Game) setStatus(s GameStatus) {
	// Only update the status when the status is different.
	if g.currentStatus != s {
		atomic.StoreInt32((*int32)(&g.currentStatus), (int32)(s))
	}
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
	}).Info("sending payload to player")
}

func (g *Game) AddPlayer(from string) {
	// If the player is being added to the game. We are going to assume
	// that he is ready to play.
	g.playersList = append(g.playersList, from)
	sort.Sort(g.playersList)
	g.playersReady.addRecvStatus(from)
}

func (g *Game) loop() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		<-ticker.C
		logrus.WithFields(logrus.Fields{
			"we":      g.listenAddr,
			"players": g.playersList,
			"status":  g.currentStatus,
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

// // getPositionOnTable return the index of our own position on the table.
// func (g *GameState) getPositionOnTable() int {
// 	for i := 0; i < len(g.playersList); i++ {
// 		if g.playersList[i].ListenAddr == g.listenAddr {
// 			return i
// 		}
// 	}

// 	panic("player does not exist in the playersList; that should not happen!!!")
// }

// func (g *GameState) getPrevPositionOnTable() int {
// 	ourPosition := g.getPositionOnTable()

// 	// if we are the in the first position on the table we need to return the last
// 	// index of the PlayersList.
// 	if ourPosition == 0 {
// 		return len(g.playersList) - 1
// 	}

// 	return ourPosition - 1
// }

// // getNextPositionOnTable returns the index of the next player in the PlayersList.
// func (g *GameState) getNextPositionOnTable() int {
// 	ourPosition := g.getPositionOnTable()

// 	// check if we are on the last position of the table, if so return first index 0.
// 	if ourPosition == len(g.playersList)-1 {
// 		return 0
// 	}

// 	return ourPosition + 1
// }

// func (g *GameState) ShuffleAndEncrypt(from string, deck [][]byte) error {
// 	g.SetPlayerStatus(from, GameStatusShuffleAndDeal)

// 	prevPlayer := g.playersList[g.getPrevPositionOnTable()]
// 	if g.isDealer && from == prevPlayer.ListenAddr {
// 		logrus.Info("shuffle round complete")
// 		return nil
// 	}

// 	dealToPlayer := g.playersList[g.getNextPositionOnTable()]

// 	logrus.WithFields(logrus.Fields{
// 		"recvFromPlayer":  from,
// 		"we":              g.listenAddr,
// 		"dealingToPlayer": dealToPlayer,
// 	}).Info("received cards and going to shuffle")

// 	// TODO:(@anthdm) encryption and shuffle
// 	// TODO: get this player out of a deterministic (sorted) list.

// 	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
// 	g.SetStatus(GameStatusShuffleAndDeal)

// 	fmt.Printf("%s => setting my own status to %s\n", g.listenAddr, GameStatusShuffleAndDeal)

// 	return nil
// }

// // InitiateShuffleAndDeal is only used for the "real" dealer. The actual "button player"
// func (g *GameState) InitiateShuffleAndDeal() {
// 	dealToPlayer := g.playersList[g.getNextPositionOnTable()]

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
