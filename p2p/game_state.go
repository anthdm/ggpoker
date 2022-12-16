package p2p

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

type GameState struct {
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
	// playersList is the list of connected players to the network
	playersList *PlayersList

	table *Table
}

func NewGame(addr string, bc chan BroadcastTo) *GameState {
	g := &GameState{
		listenAddr:          addr,
		broadcastch:         bc,
		currentStatus:       NewAtomicInt(int32(GameStatusConnected)),
		playersList:         NewPlayersList(),
		currentPlayerAction: NewAtomicInt(0),
		currentDealer:       NewAtomicInt(0),
		currentPlayerTurn:   NewAtomicInt(0),
		table:               NewTable(6),
	}

	g.playersList.add(addr)

	go g.loop()

	return g
}

func (g *GameState) canTakeAction(from string) bool {
	currentPlayerAddr := g.playersList.get(g.currentPlayerTurn.Get())
	return currentPlayerAddr == from
}

func (g *GameState) isFromCurrentDealer(from string) bool {
	return g.playersList.get(g.currentDealer.Get()) == from
}

func (g *GameState) handlePlayerAction(from string, action MessagePlayerAction) error {
	if !g.canTakeAction(from) {
		return fmt.Errorf("player (%s) taking action before his turn", from)
	}

	// If we receive a message from a peer that doenst have the same game status
	// as ours, but is not the current dealer we return an error. Cannot proceed.
	if action.CurrentGameStatus != GameStatus(g.currentStatus.Get()) && !g.isFromCurrentDealer(from) {
		return fmt.Errorf("player (%s) has not the correct game status (%s)", from, action.CurrentGameStatus)
	}

	// Every player in this case will need to set the current game status to the next one (next round).
	// NEXT
	// 1. set the next player action to IDLE
	// 2. set the next current game status
	if g.playersList.get(g.currentDealer.Get()) == from {
		g.advanceToNexRound()
	}

	// TODO:(@anthdm)
	// This still not fixed!
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

func (g *GameState) TakeAction(action PlayerAction, value int) error {
	if !g.canTakeAction(g.listenAddr) {
		return fmt.Errorf("taking action before its my turn %s", g.listenAddr)
	}

	g.currentPlayerAction.Set((int32)(action))

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

func (g *GameState) getNextGameStatus() GameStatus {
	status := GameStatus(g.currentStatus.Get())
	switch status {
	case GameStatusPreFlop:
		return GameStatusFlop
	case GameStatusFlop:
		return GameStatusTurn
	case GameStatusTurn:
		return GameStatusRiver
	case GameStatusRiver:
		return GameStatusPlayerReady
	default:
		fmt.Printf("invalid status => %+v\n", status)
		panic("invalid game status")
	}
}

func (g *GameState) advanceToNexRound() {
	g.currentPlayerAction.Set(int32(PlayerActionNone))

	if GameStatus(g.currentStatus.Get()) == GameStatusRiver {
		g.SetReady()
		return
	}
	g.currentStatus.Set(int32(g.getNextGameStatus()))
}

func (g *GameState) incNextPlayer() {
	player, err := g.table.GetPlayerAfter(g.listenAddr)
	if err != nil {
		panic(err)
	}

	if g.playersList.len()-1 == int(g.currentPlayerTurn.Get()) {
		g.currentPlayerTurn.Set(0)
		return
	}
	g.currentPlayerTurn.Inc()

	fmt.Println("the next player on the table is:", player.tablePos)
	fmt.Println("old wrong value => ", g.currentPlayerTurn)
	os.Exit(0)
}

func (g *GameState) SetStatus(s GameStatus) {
	g.setStatus(s)
	g.table.SetPlayerStatus(g.listenAddr, s)
}

func (g *GameState) setStatus(s GameStatus) {
	if s == GameStatusPreFlop {
		g.incNextPlayer()
	}

	// Only update the status when the status is different.
	if GameStatus(g.currentStatus.Get()) != s {
		g.currentStatus.Set(int32(s))
	}
}

func (g *GameState) getCurrentDealerAddr() (string, bool) {
	currentDealerAddr := g.playersList.get(g.currentDealer.Get())
	return currentDealerAddr, g.listenAddr == currentDealerAddr
}

func (g *GameState) ShuffleAndEncrypt(from string, deck [][]byte) error {
	fmt.Println("addr", g.listenAddr)
	fmt.Printf("%+v\n", g.table)
	prevPlayer, err := g.table.GetPlayerBefore(g.listenAddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("prev pos", prevPlayer)
	// [3000] == dealer
	// [5000] == small blind
	// [7000] == big blind
	if from != prevPlayer.addr {
		return fmt.Errorf("[%s] received encrypted deck from the wrong player (%s) should be (%s)", g.listenAddr, from, prevPlayer.addr)
	}

	// If we are the dealer and we received a message from
	// the previous player on the table we advance to the next round.
	_, isDealer := g.getCurrentDealerAddr()
	if isDealer && from == prevPlayer.addr {
		g.setStatus(GameStatusPreFlop)
		g.table.SetPlayerStatus(g.listenAddr, GameStatusPreFlop)
		g.sendToPlayers(MessagePreFlop{}, g.getOtherPlayers()...)
		return nil
	}

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

func (g *GameState) InitiateShuffleAndDeal() {
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

func (g *GameState) maybeDeal() {
	if GameStatus(g.currentStatus.Get()) == GameStatusPlayerReady {
		g.InitiateShuffleAndDeal()
	}
}

// SetPlayerReady is getting called when we receive a ready message
// from a player in the network taking a seat on the table.
func (g *GameState) SetPlayerReady(addr string) {
	tablePos := g.playersList.getIndex(addr)
	g.table.AddPlayerOnPosition(addr, tablePos)

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
			time.Sleep(8 * time.Second)
			g.maybeDeal()
		}()
	}
}

// SetReady is being called when we set ourselfs as ready.
func (g *GameState) SetReady() {
	tablePos := g.playersList.getIndex(g.listenAddr)
	g.table.AddPlayerOnPosition(g.listenAddr, tablePos)

	g.sendToPlayers(MessageReady{}, g.getOtherPlayers()...)
	g.setStatus(GameStatusPlayerReady)
}

func (g *GameState) sendToPlayers(payload any, addr ...string) {
	g.broadcastch <- BroadcastTo{
		To:      addr,
		Payload: payload,
	}
}

func (g *GameState) AddPlayer(from string) {
	// If the player is being added to the game. We are going to assume
	// that he is ready to play.
	g.playersList.add(from)
	sort.Sort(g.playersList)
}

func (g *GameState) loop() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		<-ticker.C

		currentDealerAddr, _ := g.getCurrentDealerAddr()
		logrus.WithFields(logrus.Fields{
			"we":  g.listenAddr,
			"pl":  g.playersList.List(),
			"gs":  GameStatus(g.currentStatus.Get()),
			"cd":  currentDealerAddr,
			"npt": g.currentPlayerTurn,
		}).Info()
	}
}

func (g *GameState) getOtherPlayers() []string {
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
func (g *GameState) getPositionOnTable() int {
	for i := 0; i < g.playersList.len(); i++ {
		if g.playersList.get(i) == g.listenAddr {
			return i
		}
	}

	panic("player does not exist in the playersList; that should not happen!!!")
}

func (g *GameState) getNextDealer() int {
	panic("TODO")
}
