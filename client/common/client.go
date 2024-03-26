package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const CONFIRMATION = "OK"

const (
	BET = uint8(iota)
	END
	WINNERS
)

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID                string
	ServerAddress     string
	LoopLapse         time.Duration
	LoopPeriod        time.Duration
	WinnersLoopPeriod time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	clientConfig ClientConfig
	readerConfig ReaderConfig
	conn         net.Conn
	done         chan bool
	reader       *BetReader
}

func (c *Client) sigterm_handler(sigterms chan os.Signal) {
	<-sigterms
	log.Infof("action: graceful_shutdown | result: in_progress | client_id: %v",
		c.clientConfig.ID,
	)

	c.done <- true

	log.Infof("action: close_socket | result: in_progress | client_id: %v",
		c.clientConfig.ID,
	)
	c.conn.Close()
	log.Infof("action: close_socket | result: success | client_id: %v",
		c.clientConfig.ID,
	)
	c.reader.Close()
	log.Infof("action: close_reader | result: success | client_id: %v",
		c.clientConfig.ID,
	)
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(clientConfig ClientConfig, readerConfig ReaderConfig) *Client {
	client := &Client{
		clientConfig: clientConfig,
		done:         make(chan bool, 1),
		readerConfig: readerConfig,
	}

	sigterms := make(chan os.Signal, 1)
	signal.Notify(sigterms, syscall.SIGTERM)
	go client.sigterm_handler(sigterms)

	return client
}

// CreateClientSocket Initializes client socket and sends the client ID.
// In case of failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.clientConfig.ServerAddress)
	if err != nil {
		log.Fatalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			err,
		)
	}

	c.conn = conn
	client_id := fmt.Sprintf("%v\n", c.clientConfig.ID)
	client_id_bytes := []byte(client_id)
	if err := c.send(client_id_bytes); err != nil {
		log.Fatalf("action: send_id | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			err,
		)
	}

	return nil
}

func (c *Client) createReader() {
	reader, err := NewReader(c.readerConfig.Path, c.readerConfig.BatchSize, c.clientConfig.ID)
	if err != nil {
		log.Fatalf("action: new_reader | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			err,
		)
	}

	c.reader = reader
}

// StartClientLoop Send batches to the client until the reader is empty
func (c *Client) StartClientLoop() error {
	c.createClientSocket()
	c.createReader()
	defer c.conn.Close()
	defer c.reader.Close()

loop:
	for bet_batch, read_err := c.reader.Read(); len(bet_batch) > 0; bet_batch, read_err = c.reader.Read() {
		if read_err != nil {
			log.Errorf("action: read_batch | result: fail | client_id: %v | error: %v",
				c.clientConfig.ID,
				read_err,
			)
			return read_err
		}

		select {
		case <-time.After(c.clientConfig.LoopPeriod):
		case <-c.done:
			break loop
		}

		if err := c.SendBatch(bet_batch); err != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | bets: %v",
				c.clientConfig.ID,
				len(bet_batch),
			)
			return err
		}
		log.Debugf("action: send_batch | result: success | client_id: %v | bets: %v", c.clientConfig.ID, len(bet_batch))

		if err := c.receiveConfirmation(); err != nil {
			log.Errorf("action: receive_confirmation | result: fail | client_id: %v | error: %v",
				c.clientConfig.ID,
				err,
			)
			return err
		}

		log.Debugf("action: receive_confirmation | result: success | client_id: %v",
			c.clientConfig.ID,
		)
	}
	if err := c.sendNotification(); err != nil {
		log.Errorf("action: send_notification | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			err,
		)
		return err
	}
	log.Infof("action: send_notification | result: success | client_id: %v",
		c.clientConfig.ID,
	)

	return nil
}

func (c *Client) sendNotification() error {
	if err := c.send([]byte{END}); err != nil {
		return err
	}

	return nil
}

func (c *Client) StartWinnersLoop() (int, error) {
	for {
		c.createClientSocket()
		defer c.conn.Close()
		log.Infof("action: get_winners | result: in_progress | client_id: %v",
			c.clientConfig.ID,
		)
		winners, err := c.GetWinners()
		if err != nil {
			if err.Error() != "EOF" {
				log.Errorf("action: get_winners | result: fail | client_id: %v | error: %v",
					c.clientConfig.ID,
					err,
				)
				return -1, err
			} else {
				select {
				case <-c.done:
					return -1, errors.New("graceful shutdown")
				case <-time.After(c.clientConfig.WinnersLoopPeriod):
					continue
				}
			}
		}

		return winners, nil
	}
}

func (c *Client) GetWinners() (int, error) {
	winner_count := 0
	for {
		if err := c.send([]byte{WINNERS}); err != nil {
			return -1, err
		}

		message_type_bytes, err := c.receive_exactly(1)
		if err != nil {
			return -1, err
		}
		message_type := uint8(message_type_bytes[0])
		if message_type == END {
			return winner_count, nil
		} else if message_type != BET {
			return -1, errors.New("unexpected message type")
		}

		length_bytes, err := c.receive_exactly(1)
		if err != nil {
			return -1, err
		}
		length := uint8(length_bytes[0])
		bet_data, err := c.receive_exactly(int(length))
		if err != nil {
			return -1, err
		}

		bet_data_str := string(bet_data)
		bet, err := DeserializeBet(bet_data_str)
		if err != nil {
			return -1, err
		}

		log.Infof("action: winner | result: success | client_id: %v | bet: %v",
			c.clientConfig.ID,
			bet,
		)
		winner_count++
	}
}

func (c *Client) receiveConfirmation() error {
	data, err := c.receive_exactly(len(CONFIRMATION))
	if err != nil {
		return err
	}

	if string(data) != CONFIRMATION {
		return errors.New("confirmation failed, unexpected response")
	}

	return nil
}

func serializeBatch(bets []Bet) ([]byte, error) {
	serialized_data := make([]byte, 0)
	for _, b := range bets {
		msg := b.Serialize()
		if len(msg) > 255 {
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

func (c *Client) SendBatch(bets []Bet) error {
	serialized_data, err := serializeBatch(bets)
	if err != nil {
		return err
	}

	batch_length := uint32(len(serialized_data))
	batch_length_bytes := make([]byte, binary.Size(batch_length))
	message_type_bytes := make([]byte, binary.Size(uint8(BET)))
	message_type_bytes[0] = uint8(BET)
	binary.BigEndian.PutUint32(batch_length_bytes, batch_length)
	metadata := append(message_type_bytes, batch_length_bytes...)
	serialized_data = append(metadata, serialized_data...)

	if err := c.send(serialized_data); err != nil {
		return err
	}

	return nil
}

func (c *Client) send(data []byte) error {
	total_bytes_written := 0
	written_bytes, err := c.conn.Write(data)
	if err != nil {
		return err
	}
	total_bytes_written += written_bytes

	for total_bytes_written < len(data) {
		if err != nil {
			return err
		}
		written_bytes, err = c.conn.Write(data[total_bytes_written:])
		total_bytes_written += written_bytes
	}

	return nil
}

func (c *Client) receive_exactly(length int) ([]byte, error) {
	data := make([]byte, length)
	buffer := make([]byte, length)
	total_bytes_read := 0
	bytes_read, err := c.conn.Read(data)
	if err != nil {
		return nil, err
	}
	total_bytes_read += bytes_read

	for total_bytes_read < length {
		bytes_read, err = c.conn.Read(buffer)
		if err != nil {
			return nil, err
		} else if bytes_read == 0 {
			return data, errors.New("EOF")
		}

		total_bytes_read += bytes_read
		data = append(data, buffer[:bytes_read]...)
	}
	return data, nil
}
