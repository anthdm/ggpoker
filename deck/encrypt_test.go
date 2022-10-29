package deck

import (
	"reflect"
	"testing"
)

func TestEncryptCard(t *testing.T) {
	key := []byte("foobarbazfoobarbazfoobarbazfoobarbaz")
	card := Card{
		Suit:  Spades,
		Value: 1,
	}

	encOutput, err := EncryptCard(key, card)
	if err != nil {
		t.Errorf("enc error %s\n", err)
	}

	decCard, err := DecryptCard(key, encOutput)
	if err != nil {
		t.Errorf("dec error %s\n", err)
	}

	if !reflect.DeepEqual(card, decCard) {
		t.Errorf("got %+v but want %+v", decCard, card)
	}
}
