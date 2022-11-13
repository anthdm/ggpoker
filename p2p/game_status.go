package p2p

type PlayerAction byte

func (pa PlayerAction) String() string {
	switch pa {
	case PlayerActionIdle:
		return "IDLE"
	case PlayerActionFold:
		return "FOLD"
	case PlayerActionCheck:
		return "CHECK"
	case PlayerActionBet:
		return "BET"
	default:
		return "INVALID"
	}
}

const (
	PlayerActionIdle PlayerAction = iota
	PlayerActionFold
	PlayerActionCheck
	PlayerActionBet
)

type GameStatus int32

func (g GameStatus) String() string {
	switch g {
	case GameStatusConnected:
		return "CONNECTED"
	case GameStatusPlayerReady:
		return "PLAYER READY"
	case GameStatusDealing:
		return "DEALING"
	case GameStatusFolded:
		return "FOLDED"
	case GameStatusChecked:
		return "CHECKED"
	case GameStatusPreFlop:
		return "PRE FLOP"
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
	GameStatusConnected GameStatus = iota
	GameStatusPlayerReady
	GameStatusDealing
	GameStatusFolded
	GameStatusChecked
	GameStatusPreFlop
	GameStatusFlop
	GameStatusTurn
	GameStatusRiver
)
