package common

import (
	"os"
	"strings"
)

const NEWLINE = "\r\n"
const READ_SIZE = 4096

type ReaderConfig struct {
	Path      string
	BatchSize int
}

type BetReader struct {
	file      *os.File
	batchSize int
	agencyID  string
	buffer    []Bet
}

func NewReader(path string, maxBatchSize int, agencyID string) (*BetReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &BetReader{
		file:      file,
		batchSize: maxBatchSize,
		agencyID:  agencyID,
		buffer:    make([]Bet, 0),
	}, nil
}

func (br *BetReader) Read() ([]Bet, error) {
	bets := make([]Bet, 0)
	if len(br.buffer) >= br.batchSize {
		bets := br.buffer[:br.batchSize]
		br.buffer = br.buffer[br.batchSize:]
		return bets, nil
	} else if len(br.buffer) > 0 {
		bets = br.buffer
		br.buffer = make([]Bet, 0)
	}

	betsFromFile, err := br.readFromFile()
	if err != nil {
		return nil, err
	}

	bets = append(bets, betsFromFile...)

	if len(bets) > br.batchSize {
		br.buffer = append(br.buffer, bets[br.batchSize:]...)
		bets = bets[:br.batchSize]
	}

	return bets, nil
}

func (br *BetReader) readFromFile() ([]Bet, error) {
	bets := make([]Bet, 0)
	data := make([]byte, READ_SIZE)
	read_bytes, err := br.file.Read(data)
	if err != nil {
		return nil, err
	}

	for i := read_bytes - 1; i >= 0; i-- {
		if data[i] == '\n' {
			data = data[:i]
			br.file.Seek(int64(i-read_bytes), 1)
			break
		}
	}

	for _, line := range strings.Split(string(data), NEWLINE) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		bet, err := BetFromCSV(line)
		if err != nil {
			return nil, err
		}

		bets = append(bets, bet)
	}

	return bets, nil
}

func (br *BetReader) Close() error {
	return br.file.Close()
}
