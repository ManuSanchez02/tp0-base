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

// Read reads a batch of bets from the file.
// The function will return at most, the number of bets specified in the batch size.
func (br *BetReader) Read() ([]Bet, error) {
	bets := make([]Bet, 0)
	if len(br.buffer) >= br.batchSize {
		bets = br.buffer[:br.batchSize]
		br.buffer = br.buffer[br.batchSize:]
		return bets, nil
	} else if len(br.buffer) > 0 {
		bets = br.buffer
		br.buffer = make([]Bet, 0)
	}

	for len(bets) < br.batchSize {
		betsFromFile, err := br.readFromFile()
		if err != nil {
			if err.Error() == "EOF" {
				bets = append(bets, betsFromFile...)
				break
			}
			return nil, err
		}

		bets = append(bets, betsFromFile...)
	}

	if len(bets) > br.batchSize {
		br.buffer = append(br.buffer, bets[br.batchSize:]...)
		bets = bets[:br.batchSize]
	}

	return bets, nil
}

// Read bets from the file and return them as a slice of bets.
// It will read chunks of READ_SIZE bytes from the file and split them by newline.
// If a line is not complete, it will perform a seek to the correct position in the file.
func (br *BetReader) readFromFile() ([]Bet, error) {
	data := make([]byte, READ_SIZE)
	total_read_bytes, err := br.file.Read(data)
	if err != nil {
		return nil, err
	}

	if total_read_bytes < READ_SIZE {
		// Retry reading once
		read_bytes, read_err := br.file.Read(data[total_read_bytes:])
		if read_err != nil {
			if read_err.Error() == "EOF" {
				bets, parse_err := parseBets(data[:total_read_bytes+read_bytes])
				if parse_err != nil {
					return nil, parse_err
				}

				return bets, parse_err
			}

			return nil, read_err
		}
		total_read_bytes += read_bytes
	}

	for i := total_read_bytes - 1; i >= 0; i-- {
		if data[i] == '\n' {
			data = data[:i]
			br.file.Seek(int64(i-total_read_bytes), 1)
			break
		}
	}

	return parseBets(data)
}

func (br *BetReader) Close() error {
	return br.file.Close()
}

func parseBets(data []byte) ([]Bet, error) {
	bets := make([]Bet, 0)
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
