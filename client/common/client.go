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

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopLapse     time.Duration
	LoopPeriod    time.Duration
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
	c.reader.Close()
	log.Infof("action: close_socket | result: success | client_id: %v",
		c.clientConfig.ID,
	)
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(clientConfig ClientConfig, readerConfig ReaderConfig) *Client {
	client := &Client{
		clientConfig: clientConfig,
		done:         make(chan bool),
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
	written_bytes, err := fmt.Fprintf(c.conn, "%v\n", c.clientConfig.ID)
	if err != nil {
		log.Fatalf("action: send_id | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			err,
		)
	} else if written_bytes != len(c.clientConfig.ID)+1 {
		log.Fatalf("action: send_id | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			errors.New("short-write"),
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
func (c *Client) StartClientLoop() {
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
			return
		}

		select {
		case <-time.After(c.clientConfig.LoopPeriod):
		case <-c.done:
			break loop
		}

		if c.SendBatch(bet_batch) != nil {
			log.Errorf("action: send_batch | result: fail | client_id: %v | bets: %v",
				c.clientConfig.ID,
				len(bet_batch),
			)
			return
		}
		log.Infof("action: send_batch | result: success | client_id: %v | bets: %v", c.clientConfig.ID, len(bet_batch))

		if c.receiveConfirmation() != nil {
			log.Errorf("action: receive_confirmation | result: fail | client_id: %v | error: %v",
				c.clientConfig.ID,
				read_err,
			)
			return
		}

		log.Infof("action: receive_confirmation | result: success | client_id: %v",
			c.clientConfig.ID,
		)
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.clientConfig.ID)
}

func (c *Client) ProcessBet(bet Bet) {
	c.createClientSocket()
	defer c.conn.Close()

	msg, send_err := c.sendBet(bet)
	if send_err != nil {
		log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			send_err,
		)
		return
	}

	log.Infof("action: send_bet | result: success | client_id: %v | msg: %v",
		c.clientConfig.ID,
		msg,
	)

	if receive_err := c.receiveConfirmation(); receive_err != nil {
		log.Errorf("action: receive_confirmation | result: fail | client_id: %v | error: %v",
			c.clientConfig.ID,
			receive_err,
		)
		return
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
		bet.Document,
		bet.Number,
	)
}

func (c *Client) sendBet(bet Bet) (string, error) {
	msg := bet.Serialize()
	if len(msg) > 255 {
		return "", errors.New("message too long")
	}
	length := uint8(len(msg))
	length_bytes := make([]byte, binary.Size(length))
	length_bytes[0] = length
	total_bytes_written := 0
	for bytes_written, err := c.conn.Write(length_bytes); total_bytes_written < len(length_bytes); bytes_written, err = c.conn.Write(length_bytes) {
		if err != nil {
			return "", err
		}
		total_bytes_written += bytes_written
	}

	total_bytes_written = 0
	for bytes_written, err := fmt.Fprintf(c.conn, "%v", msg); total_bytes_written < len(msg); bytes_written, err = fmt.Fprintf(c.conn, "%v", msg) {
		if err != nil {
			return "", err
		}
		total_bytes_written += bytes_written
	}

	return msg, nil
}

func (c *Client) receiveConfirmation() error {
	response := make([]byte, len(CONFIRMATION))
	total_bytes_read := 0
	for bytes_read, err := c.conn.Read(response); total_bytes_read < len(CONFIRMATION); bytes_read, err = c.conn.Read(response) {
		if err != nil {
			return err
		} else if string(response) != CONFIRMATION {
			return errors.New("unexpected response")
		}

		total_bytes_read += bytes_read
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
	binary.BigEndian.PutUint32(batch_length_bytes, batch_length)
	serialized_data = append(batch_length_bytes, serialized_data...)

	total_bytes_written := 0
	for written_bytes, err := c.conn.Write(serialized_data); total_bytes_written < len(serialized_data); written_bytes, err = c.conn.Write(serialized_data) {
		if err != nil {
			return err
		}
		total_bytes_written += written_bytes
	}

	return nil
}
