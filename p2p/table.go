package p2p

import (
	"fmt"
	"sync"
)

type Player struct {
	addr          string
	currentAction PlayerAction
	gameStatus    GameStatus
	tablePos      int
}

func NewPlayer(addr string) *Player {
	return &Player{
		addr:          addr,
		currentAction: PlayerActionNone,
		gameStatus:    GameStatusConnected,
		tablePos:      -1,
	}
}

type Table struct {
	lock  sync.RWMutex
	seats map[int]*Player

	maxSeats int
}

func NewTable(maxSeats int) *Table {
	return &Table{
		seats:    make(map[int]*Player),
		maxSeats: maxSeats,
	}
}

// TODO: (@anthdm) !!
func (t *Table) String() string {
	return ""
}

func (t *Table) Players() []*Player {
	t.lock.RLock()
	defer t.lock.RUnlock()

	players := []*Player{}
	for i := 0; i < t.maxSeats; i++ {
		player, ok := t.seats[i]
		if ok {
			players = append(players, player)
		}
	}

	return players
}

func (t *Table) GetPlayerBefore(addr string) (*Player, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	currentPlayer, err := t.getPlayer(addr)
	if err != nil {
		return nil, err
	}

	i := currentPlayer.tablePos - 1
	for {
		prevPlayer, ok := t.seats[i]
		if prevPlayer == currentPlayer {
			return nil, fmt.Errorf("%s is the only player on the table", addr)
		}
		if ok {
			return prevPlayer, nil
		}

		i--
		if i <= 0 {
			i = t.maxSeats
		}
	}
}

func (t *Table) GetPlayerAfter(addr string) (*Player, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	currentPlayer, err := t.getPlayer(addr)
	if err != nil {
		return nil, err
	}

	i := currentPlayer.tablePos + 1
	for {
		nextPlayer, ok := t.seats[i]
		if nextPlayer == currentPlayer {
			return nil, fmt.Errorf("%s is the only player on the table", addr)
		}
		if ok {
			return nextPlayer, nil
		}

		i++
		if t.maxSeats <= i {
			i = 0
		}
	}
}

func (t *Table) clear() {
	t.seats = map[int]*Player{}
}

func (t *Table) LenPlayers() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.seats)
}

func (t *Table) RemovePlayerByAddr(addr string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	for i := 0; i < t.maxSeats; i++ {
		player, ok := t.seats[i]
		if ok {
			if player.addr == addr {
				delete(t.seats, i)
				return nil
			}
		}
	}

	return fmt.Errorf("player (%s) not on the table", addr)
}

func (t *Table) GetPlayer(addr string) (*Player, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.getPlayer(addr)
}

func (t *Table) getPlayer(addr string) (*Player, error) {
	for i := 0; i < t.maxSeats; i++ {
		player, ok := t.seats[i]
		if ok {
			if player.addr == addr {
				return player, nil
			}
		}
	}

	return nil, fmt.Errorf("player (%s) not on the table", addr)
}

func (t *Table) AddPlayer(addr string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(t.seats) == t.maxSeats {
		return fmt.Errorf("player table is full")
	}

	pos := t.getNextFreeSeat()
	player := NewPlayer(addr)
	player.tablePos = pos

	t.seats[pos] = player

	return nil
}

func (t *Table) getNextFreeSeat() int {
	for i := 0; i < t.maxSeats; i++ {
		if _, ok := t.seats[i]; !ok {
			return i
		}
	}

	panic("no free seat is available!!")
}
