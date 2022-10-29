package deck

import (
	"fmt"
	"math/rand"
	"strconv"
)

type Suit int

func (s Suit) String() string {
	switch s {
	case Spades:
		return "SPADES"
	case Harts:
		return "HARTS"
	case Diamonds:
		return "DIAMONDS"
	case Clubs:
		return "CLUBS"
	default:
		panic("invalid card suit")
	}
}

const (
	Spades   Suit = iota // 0
	Harts                // 1
	Diamonds             // 2
	Clubs                // 3
)

type Card struct {
	Suit  Suit
	Value int
}

func (c Card) String() string {
	value := strconv.Itoa(c.Value)
	if c.Value == 1 {
		value = "ACE"
	}

	return fmt.Sprintf("%s of %s %s", value, c.Suit, suitToUnicode(c.Suit))
}

func NewCard(s Suit, v int) Card {
	if v > 13 {
		panic("the value of the card cannot be higher then 13")
	}

	return Card{
		Suit:  s,
		Value: v,
	}
}

type Deck [52]Card

func New() Deck {
	var (
		nSuits = 4
		nCards = 13
		d      = [52]Card{}
	)

	x := 0
	for i := 0; i < nSuits; i++ {
		for j := 0; j < nCards; j++ {
			d[x] = NewCard(Suit(i), j+1)
			x++
		}
	}

	return shuffle(d)
}

func shuffle(d Deck) Deck {
	for i := 0; i < len(d); i++ {
		r := rand.Intn(i + 1)

		if r != i {
			d[i], d[r] = d[r], d[i]
		}
	}

	return d
}

func suitToUnicode(s Suit) string {
	switch s {
	case Spades:
		return "♠"
	case Harts:
		return "♥"
	case Diamonds:
		return "♦"
	case Clubs:
		return "♣"
	default:
		panic("invalid card suit")
	}
}
