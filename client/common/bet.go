package common

import (
	"strconv"
	"strings"
)

const DELIMITER = ";"

type Bet struct {
	AgencyID  string
	Document  int
	BirthDate string
	Number    int
	FirstName string
	LastName  string
}

func (b Bet) Serialize() string {
	info := [6]string{b.AgencyID, b.FirstName, b.LastName, strconv.Itoa(b.Document), b.BirthDate, strconv.Itoa(b.Number)}
	msg := strings.Join(info[:], DELIMITER)
	return msg
}
