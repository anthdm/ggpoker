package p2p

type Round uint32

const (
	Dealing Round = iota
	PreFlop
	Flop
	Turn
	River
)

type GameState struct {
	isDealer bool   // atomic accessable
	Round    uint32 // atomic accessable
}

func NewGameState() *GameState {
	return &GameState{}
}

func (g *GameState) loop() {

}
