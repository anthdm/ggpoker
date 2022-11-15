package p2p

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableGetPlayerBefore(t *testing.T) {
	var (
		maxSeats = 6
		table    = NewTable(maxSeats)
	)

	assert.Nil(t, table.AddPlayer("1"))
	assert.Nil(t, table.AddPlayer("2"))
	prevPlayer, err := table.GetPlayerBefore("2")
	assert.Nil(t, err)
	assert.Equal(t, prevPlayer.addr, "1")

	prevPlayer, err = table.GetPlayerBefore("1")
	assert.Nil(t, err)
	assert.Equal(t, prevPlayer.addr, "2")
}

func TestTableGetPlayerAfter(t *testing.T) {
	var (
		maxSeats = 10
		table    = NewTable(maxSeats)
	)

	assert.Nil(t, table.AddPlayer("1"))
	assert.Nil(t, table.AddPlayer("2"))
	nextPlayer, err := table.GetPlayerAfter("1")
	assert.Nil(t, err)
	assert.Equal(t, nextPlayer.addr, "2")

	assert.Nil(t, table.AddPlayer("3"))
	assert.Nil(t, table.RemovePlayerByAddr("2"))
	nextPlayer, err = table.GetPlayerAfter("1")
	assert.Nil(t, err)
	assert.Equal(t, nextPlayer.addr, "3")

	assert.Nil(t, table.RemovePlayerByAddr("3"))
	nextPlayer, err = table.GetPlayerAfter("1")
	assert.NotNil(t, err)
	assert.Nil(t, nextPlayer)

	assert.Nil(t, table.AddPlayer("2"))
	nextPlayer, err = table.GetPlayerAfter("2")
	assert.Nil(t, err)
	assert.Equal(t, nextPlayer.addr, "1")

	// Test the edge case on the last player
	table.clear()
	table.maxSeats = 3
	assert.Nil(t, table.AddPlayer("1"))
	assert.Nil(t, table.AddPlayer("2"))
	assert.Nil(t, table.AddPlayer("3"))

	nextPlayer, err = table.GetPlayerAfter("3")
	assert.Nil(t, err)
	assert.NotNil(t, nextPlayer)
	assert.Equal(t, nextPlayer.addr, "1")
}

func TestTableRemovePlayer(t *testing.T) {
	var (
		maxSeats = 10
		table    = NewTable(maxSeats)
	)

	for i := 0; i < maxSeats; i++ {
		addr := fmt.Sprintf("%d", i)
		assert.Nil(t, table.AddPlayer(addr))
		assert.Nil(t, table.RemovePlayerByAddr(addr))

		player, err := table.GetPlayer(addr)
		assert.NotNil(t, err)
		assert.Nil(t, player)
	}
}

func TestTableAddPlayer(t *testing.T) {
	var (
		maxSeats = 2
		table    = NewTable(maxSeats)
	)

	assert.Nil(t, table.AddPlayer(":1"))
	assert.Nil(t, table.AddPlayer(":2"))

	assert.Equal(t, 2, table.LenPlayers())

	assert.NotNil(t, table.AddPlayer(":3"))
	assert.Equal(t, 2, table.LenPlayers())
}

func TestTableGetPlayer(t *testing.T) {
	var (
		maxSeats = 10
		table    = NewTable(maxSeats)
	)

	for i := 0; i < maxSeats; i++ {
		addr := fmt.Sprintf("%d", i)
		assert.Nil(t, table.AddPlayer(addr))
		player, err := table.GetPlayer(addr)
		assert.Nil(t, err)
		assert.Equal(t, player.addr, addr)
	}
	assert.Equal(t, maxSeats, table.LenPlayers())
}
