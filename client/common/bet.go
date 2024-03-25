package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const PROTOCOL_DELIMITER = ";"
const CSV_DELIMITER = ","
const MAX_MESSAGE_SIZE = 255

type Bet struct {
	Document  int
	BirthDate string
	Number    int
	FirstName string
	LastName  string
}

func (b Bet) Serialize() string {
	info := [5]string{b.FirstName, b.LastName, strconv.Itoa(b.Document), b.BirthDate, strconv.Itoa(b.Number)}
	msg := strings.Join(info[:], PROTOCOL_DELIMITER)
	return msg
}

func BetFromCSV(data string) (Bet, error) {
	info := strings.Split(data, CSV_DELIMITER)
	if len(info) != 5 {
		errorString := fmt.Sprintf("invalid data: %s", data)
		return Bet{}, errors.New(errorString)
	}
	document, err := strconv.Atoi(info[2])
	if err != nil {
		return Bet{}, err
	}

	number, err := strconv.Atoi(info[4])
	if err != nil {
		return Bet{}, err
	}

	return Bet{
		FirstName: info[0],
		LastName:  info[1],
		Document:  document,
		BirthDate: info[3],
		Number:    number,
	}, nil
}

func SerializeBatch(bets []Bet) ([]byte, error) {
	serialized_data := make([]byte, 0)
	for _, b := range bets {
		msg := b.Serialize()
		if len(msg) > MAX_MESSAGE_SIZE {
			return nil, errors.New("message too long")
		}

		length := uint8(len(msg))
		length_bytes := make([]byte, binary.Size(length))
		length_bytes[0] = length

		serialized_data = append(serialized_data, length_bytes...)
		serialized_data = append(serialized_data, []byte(msg)...)
	}

	return serialized_data, nil
}
