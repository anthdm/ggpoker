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

type PlayerAction byte

const (
	PlayerActionFold  PlayerAction = iota + 1 // 1
	PlayerActionCheck                         // 2
	PlayerActionBet                           // 3
)

type PlayerActionsRecv struct {
	mu          sync.RWMutex
	recvActions map[string]MessagePlayerAction
}

func NewPlayerActionsRevc() *PlayerActionsRecv {
	return &PlayerActionsRecv{
		recvActions: make(map[string]MessagePlayerAction),
	}
}

func (pa *PlayerActionsRecv) addAction(from string, action MessagePlayerAction) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.recvActions[from] = action
}

// TODO: (@anthdm) Maybe use playersList instead??
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
	// currentDealer should be atomically accessable.
	// NOTE: this will be -1 when the game is in a bootstrapped state.
	currentDealer int32
	// currentPlayerTurn should be atomically accessable.
	currentPlayerTurn int32

	playersReady      *PlayersReady
	recvPlayerActions *PlayerActionsRecv

	playersList PlayersList
}

func NewGame(addr string, bc chan BroadcastTo) *Game {
	g := &Game{
		listenAddr:        addr,
		broadcastch:       bc,
		currentStatus:     GameStatusConnected,
		playersReady:      NewPlayersReady(),
		playersList:       PlayersList{},
		currentDealer:     0,
		recvPlayerActions: NewPlayerActionsRevc(),
	}

	g.playersList = append(g.playersList, addr)

	go g.loop()

	return g
}

func (g *Game) setCurrentPlayerTurn(index int32) {
	atomic.StoreInt32(&g.currentPlayerTurn, index)
}

func (g *Game) canTakeAction(from string) bool {
	currentPlayerAddr := g.playersList[g.currentPlayerTurn]

	return currentPlayerAddr == from
}

func (g *Game) handlePlayerAction(from string, action MessagePlayerAction) error {
	if !g.canTakeAction(from) {
		return fmt.Errorf("player (%s) taking action before his turn", from)
	}

	logrus.WithFields(logrus.Fields{
		"we":   g.listenAddr,
		"from": from,
	}).Info("recv player action")

	g.recvPlayerActions.addAction(from, action)

	// TODO: (@anthdm) This function should be handle the logic of picking the next player
	// internally. Cause maybe the next player in the list in not the next index, hence not
	// g.currentPlayerTurn + 1, due to the fact that his status can be just "connected"
	g.setCurrentPlayerTurn(g.currentPlayerTurn + 1)

	return nil
}

func (g *Game) Fold() {
	g.SetStatus(GameStatusFolded)

	// TODO:(@anthdm) Make a wrapper function that can be used for each PlayerAction
	// (fold, check, ...)
	g.setCurrentPlayerTurn(g.currentPlayerTurn + 1)

	action := MessagePlayerAction{
		Action:            PlayerActionFold,
		CurrentGameStatus: g.currentStatus,
	}

	g.sendToPlayers(action, g.getOtherPlayers()...)
}

func (g *Game) SetStatus(s GameStatus) {
	g.setStatus(s)
}

func (g *Game) setStatus(s GameStatus) {
	if s == GameStatusPreFlop {
		g.setCurrentPlayerTurn(g.currentDealer + 1)
	}

	// Only update the status when the status is different.
	if g.currentStatus != s {
		atomic.StoreInt32((*int32)(&g.currentStatus), (int32)(s))
	}
}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealerAddr := g.playersList[g.currentDealer]

	return currentDealerAddr, g.listenAddr == currentDealerAddr
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
		g.setStatus(GameStatusPreFlop)
		g.sendToPlayers(MessagePreFlop{}, g.getOtherPlayers()...)
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
			"we":             g.listenAddr,
			"players":        g.playersList,
			"status":         g.currentStatus,
			"currentDealer":  currentDealerAddr,
			"nextPlayerTurn": g.currentPlayerTurn,
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
