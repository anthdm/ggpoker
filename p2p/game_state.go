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

type PlayersList struct {
	lock sync.RWMutex
	list []string
}

func NewPlayersList() *PlayersList {
	return &PlayersList{
		list: []string{},
	}
}

func (p *PlayersList) List() []string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.list
}

func (p *PlayersList) len() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.list)
}

func (p *PlayersList) add(addr string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.list = append(p.list, addr)
	sort.Sort(p)
}

func (p *PlayersList) getIndex(addr string) int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for i := 0; i < len(p.list); i++ {
		if addr == p.list[i] {
			return i
		}
	}
	return -1
}

func (p *PlayersList) get(index any) string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var i int
	switch v := index.(type) {
	case int:
		i = v
	case int32:
		i = int(v)
	}

	if len(p.list)-1 < i {
		panic("the given index is too high")
	}

	return p.list[i]
}

func (p *PlayersList) Len() int { return len(p.list) }
func (p *PlayersList) Swap(i, j int) {
	p.list[i], p.list[j] = p.list[j], p.list[i]
}
func (p *PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(p.list[i][1:])
	portJ, _ := strconv.Atoi(p.list[j][1:])

	return portI < portJ
}

type AtomicInt struct {
	value int32
}

func NewAtomicInt(value int32) *AtomicInt {
	return &AtomicInt{
		value: value,
	}
}

func (a *AtomicInt) String() string {
	return fmt.Sprintf("%d", a.value)
}

func (a *AtomicInt) Set(value int32) {
	atomic.StoreInt32(&a.value, value)
}

func (a *AtomicInt) Get() int32 {
	return atomic.LoadInt32(&a.value)
}

func (a *AtomicInt) Inc() {
	currentValue := a.Get()
	a.Set(currentValue + 1)
}

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

func (pa *PlayerActionsRecv) clear() {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.recvActions = map[string]MessagePlayerAction{}
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
	currentStatus *AtomicInt

	// currentPlayerAction should be atomically accessable.
	currentPlayerAction *AtomicInt
	// currentDealer should be atomically accessable.
	// NOTE: this will be -1 when the game is in a bootstrapped state.
	currentDealer *AtomicInt
	// currentPlayerTurn should be atomically accessable.
	currentPlayerTurn *AtomicInt

	playersReady      *PlayersReady
	recvPlayerActions *PlayerActionsRecv

	// playersList is the list of connected players to the network
	playersList *PlayersList

	table *Table
}

func NewGame(addr string, bc chan BroadcastTo) *Game {
	g := &Game{
		listenAddr:          addr,
		broadcastch:         bc,
		currentStatus:       NewAtomicInt(int32(GameStatusConnected)),
		playersReady:        NewPlayersReady(),
		playersList:         NewPlayersList(),
		currentPlayerAction: NewAtomicInt(0),
		currentDealer:       NewAtomicInt(0),
		recvPlayerActions:   NewPlayerActionsRevc(),
		currentPlayerTurn:   NewAtomicInt(0),
		table:               NewTable(6),
	}

	g.playersList.add(addr)

	go g.loop()

	return g
}

func (g *Game) getNextGameStatus() GameStatus {
	switch GameStatus(g.currentStatus.Get()) {
	case GameStatusPreFlop:
		return GameStatusFlop
	case GameStatusFlop:
		return GameStatusTurn
	case GameStatusTurn:
		return GameStatusRiver
	default:
		panic("invalid game status")
	}
}

func (g *Game) canTakeAction(from string) bool {
	currentPlayerAddr := g.playersList.get(g.currentPlayerTurn.Get())
	return currentPlayerAddr == from
}

func (g *Game) isFromCurrentDealer(from string) bool {
	return g.playersList.get(g.currentDealer.Get()) == from
}

func (g *Game) handlePlayerAction(from string, action MessagePlayerAction) error {
	if !g.canTakeAction(from) {
		return fmt.Errorf("player (%s) taking action before his turn", from)
	}

	// If we receive a message from a peer that doenst have the same game status
	// as ours, but is not the current dealer we return an error. Cannot proceed.
	if action.CurrentGameStatus != GameStatus(g.currentStatus.Get()) && !g.isFromCurrentDealer(from) {
		return fmt.Errorf("player (%s) has not the correct game status (%s)", from, action.CurrentGameStatus)
	}

	g.recvPlayerActions.addAction(from, action)

	// Every player in this case will need to set the current game status to the next one (next round).
	// NEXT
	// 1. set the next player action to IDLE
	// 2. set the next current game status
	if g.playersList.get(g.currentDealer.Get()) == from {
		g.advanceToNexRound()
	}

	// TODO:(@anthdm)
	// This still not fixed!
	// This function should be handle the logic of picking the next player
	// internally. Cause maybe the next player in the list in not the next index, hence not
	// g.currentPlayerTurn + 1, due to the fact that his status can be just "connected"
	g.incNextPlayer()

	logrus.WithFields(logrus.Fields{
		"we":     g.listenAddr,
		"from":   from,
		"action": action,
	}).Info("recv player action")

	return nil
}

func (g *Game) TakeAction(action PlayerAction, value int) error {
	if !g.canTakeAction(g.listenAddr) {
		return fmt.Errorf("taking action before its my turn %s", g.listenAddr)
	}

	g.currentPlayerAction.Set((int32)(action))
	// if action == PlayerActionFold {
	// 	//
	// }
	// if action == PlayerActionCheck {
	// 	//
	// }

	g.incNextPlayer()

	// If we are the dealer that just took an action, we can go to the next round.
	if g.listenAddr == g.playersList.get(g.currentDealer.Get()) {
		// NEXT
		g.advanceToNexRound()
	}

	a := MessagePlayerAction{
		Action:            action,
		CurrentGameStatus: GameStatus(g.currentStatus.Get()),
		Value:             value,
	}
	g.sendToPlayers(a, g.getOtherPlayers()...)

	return nil
}

func (g *Game) advanceToNexRound() {
	/// g.currentDealer.Set()
	g.recvPlayerActions.clear()
	g.currentPlayerAction.Set(int32(PlayerActionNone))
	g.currentStatus.Set(int32(g.getNextGameStatus()))
}

func (g *Game) incNextPlayer() {
	if g.playersList.len()-1 == int(g.currentPlayerTurn.Get()) {
		g.currentPlayerTurn.Set(0)
		return
	}

	g.currentPlayerTurn.Inc()
}

func (g *Game) SetStatus(s GameStatus) {
	g.setStatus(s)
	g.table.SetPlayerStatus(g.listenAddr, s)
}

func (g *Game) setStatus(s GameStatus) {
	if s == GameStatusPreFlop {
		g.incNextPlayer()
	}

	// Only update the status when the status is different.
	if GameStatus(g.currentStatus.Get()) != s {
		g.currentStatus.Set(int32(s))
	}
}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealerAddr := g.playersList.get(g.currentDealer.Get())

	return currentDealerAddr, g.listenAddr == currentDealerAddr
}

func (g *Game) ShuffleAndEncrypt(from string, deck [][]byte) error {
	prevPlayer, err := g.table.GetPlayerBefore(g.listenAddr)
	if err != nil {
		panic(err)
	}
	if from != prevPlayer.addr {
		return fmt.Errorf("received encrypted deck from the wrong player (%s) should be (%s)", from, prevPlayer.addr)
	}

	// If we are the dealer and we received a message from the previous player on the table
	// we advance to the next round.
	_, isDealer := g.getCurrentDealerAddr()
	if isDealer && from == prevPlayer.addr {
		g.setStatus(GameStatusPreFlop)
		g.table.SetPlayerStatus(g.listenAddr, GameStatusPreFlop)
		g.sendToPlayers(MessagePreFlop{}, g.getOtherPlayers()...)
		return nil
	}

	// dealToPlayer := g.playersList.get(g.getNextPositionOnTable())
	dealToPlayer, err := g.table.GetPlayerAfter(g.listenAddr)
	if err != nil {
		panic(err)
	}

	logrus.WithFields(logrus.Fields{
		"recvFromPlayer":  from,
		"we":              g.listenAddr,
		"dealingToPlayer": dealToPlayer.addr,
	}).Info("received cards and going to shuffle")

	// TODO:(@anthdm) encryption and shuffle
	g.sendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayer.addr)
	g.setStatus(GameStatusDealing)

	return nil
}

func (g *Game) InitiateShuffleAndDeal() {
	fmt.Println("==================================================================")
	fmt.Println(g.listenAddr)
	fmt.Println("==================================================================")

	// dealToPlayerAddr := g.playersList.get(g.getNextPositionOnTable())
	dealToPlayer, err := g.table.GetPlayerAfter(g.listenAddr)
	if err != nil {
		panic(err)
	}

	g.setStatus(GameStatusDealing)
	g.sendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayer.addr)

	logrus.WithFields(logrus.Fields{
		"we": g.listenAddr,
		"to": dealToPlayer.addr,
	}).Info("dealing cards")
}

func (g *Game) maybeDeal() {
	if GameStatus(g.currentStatus.Get()) == GameStatusPlayerReady {
		g.InitiateShuffleAndDeal()
	}
}

// SetPlayerReady is getting called when we receive a ready message
// from a player in the network.
func (g *Game) SetPlayerReady(addr string) {
	tablePos := g.playersList.getIndex(addr)
	g.table.AddPlayerOnPosition(addr, tablePos)

	// TODO: (@anthdm) check if we really need this??
	g.playersReady.addRecvStatus(addr)

	// TODO(@anthdm): This potentially going to cause an issue!
	// If we don't have enough players the round cannot be started.
	if g.table.LenPlayers() < 2 {
		return
	}

	// we need to check if we are the dealer of the current round.
	if _, areWeDealer := g.getCurrentDealerAddr(); areWeDealer {
		go func() {
			// if the game can start we will wait another
			// N amount of seconds to actually start dealing
			time.Sleep(5 * time.Second)
			g.maybeDeal()
		}()
	}
}

// SetReady is being called when we set ourselfs as ready.
func (g *Game) SetReady() {
	tablePos := g.playersList.getIndex(g.listenAddr)
	g.table.AddPlayerOnPosition(g.listenAddr, tablePos)

	g.playersReady.addRecvStatus(g.listenAddr)
	g.sendToPlayers(MessageReady{}, g.getOtherPlayers()...)
	g.setStatus(GameStatusPlayerReady)
}

func (g *Game) sendToPlayers(payload any, addr ...string) {
	g.broadcastch <- BroadcastTo{
		To:      addr,
		Payload: payload,
	}
}

func (g *Game) AddPlayer(from string) {
	// If the player is being added to the game. We are going to assume
	// that he is ready to play.
	g.playersList.add(from)
	sort.Sort(g.playersList)
}

func (g *Game) loop() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		<-ticker.C

		currentDealerAddr, _ := g.getCurrentDealerAddr()
		logrus.WithFields(logrus.Fields{
			"we":             g.listenAddr,
			"playersList":    g.playersList.List(),
			"gameState":      GameStatus(g.currentStatus.Get()),
			"currentDealer":  currentDealerAddr,
			"nextPlayerTurn": g.currentPlayerTurn,
		}).Info()
	}
}

func (g *Game) getOtherPlayers() []string {
	players := []string{}

	for _, addr := range g.playersList.List() {
		if addr == g.listenAddr {
			continue
		}
		players = append(players, addr)
	}

	return players
}

// getPositionOnTable return the index of our own position on the table.
func (g *Game) getPositionOnTable() int {
	for i := 0; i < g.playersList.len(); i++ {
		if g.playersList.get(i) == g.listenAddr {
			return i
		}
	}

	panic("player does not exist in the playersList; that should not happen!!!")
}

// func (g *Game) getPrevPositionOnTable() int {
// 	ourPosition := g.getPositionOnTable()

// 	// if we are the in the first position on the table we need to return the last
// 	// index of the PlayersList.
// 	if ourPosition == 0 {
// 		return g.playersList.len() - 1
// 	}

// 	return ourPosition - 1
// }

func (g *Game) getNextDealer() int {
	panic("TODO")
}

// getNextPositionOnTable returns the index of the next player in the PlayersList.
// func (g *Game) getNextPositionOnTable() int {
// 	ourPosition := g.getPositionOnTable()

// 	// check if we are on the last position of the table, if so return first index 0.
// 	if ourPosition == g.playersList.len()-1 {
// 		return 0
// 	}

// 	return ourPosition + 1
// }
