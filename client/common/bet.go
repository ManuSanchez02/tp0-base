package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const PROTOCOL_DELIMITER = ";"
const CSV_DELIMITER = ","

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
