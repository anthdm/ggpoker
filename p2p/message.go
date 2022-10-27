package p2p

type Message struct {
	Payload any
	From    string
}

type BroadcastTo struct {
	To      []string
	Payload any
}

func NewMessage(from string, payload any) *Message {
	return &Message{
		From:    from,
		Payload: payload,
	}
}

type Handshake struct {
	Version     string
	GameVariant GameVariant
	GameStatus  GameStatus
	ListenAddr  string
}

type MessagePeerList struct {
	Peers []string
}

type MessageEncDeck struct {
	Deck [][]byte
}
