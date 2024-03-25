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

func (b Bet) String() string {
	return fmt.Sprintf("(document: %d | number: %d)", b.Document, b.Number)
}

func DeserializeBet(data string) (Bet, error) {
	info := strings.Split(data, PROTOCOL_DELIMITER)
	return betFromStringArray(info)
}

func BetFromCSV(data string) (Bet, error) {
	info := strings.Split(data, CSV_DELIMITER)
	return betFromStringArray(info)
}

func betFromStringArray(data []string) (Bet, error) {
	if len(data) != 5 {
		errorString := fmt.Sprintf("invalid data: %s", data)
		return Bet{}, errors.New(errorString)
	}
	document, err := strconv.Atoi(data[2])
	if err != nil {
		return Bet{}, err
	}

	number, err := strconv.Atoi(data[4])
	if err != nil {
		return Bet{}, err
	}

	return Bet{
		FirstName: data[0],
		LastName:  data[1],
		Document:  document,
		BirthDate: data[3],
		Number:    number,
	}, nil
}
