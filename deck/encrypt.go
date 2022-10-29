package deck

import (
	"bytes"
	"encoding/gob"
)

func DecryptCard(key, encCard []byte) (Card, error) {
	card := Card{}

	b, err := Encrypt(key, encCard)
	if err != nil {
		return card, err
	}

	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&card); err != nil {
		return card, err
	}

	return card, nil
}

func EncryptCard(key []byte, card Card) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(card); err != nil {
		return nil, err
	}
	payload := buf.Bytes()

	return Encrypt(key, payload)
}

func Encrypt(key, payload []byte) ([]byte, error) {
	encOutput := make([]byte, len(payload))
	for i := 0; i < len(payload); i++ {
		encOutput[i] = payload[i] ^ key[i%len(key)]
	}

	return encOutput, nil
}
