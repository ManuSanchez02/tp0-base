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

func (c *Client) sigtermHandler(sigterms chan os.Signal) {
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
	go client.sigtermHandler(sigterms)

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

// Receives a batch confirmation from the server indicating that the batch was
// successfully received
func (c *Client) receiveConfirmation() error {
	data, err := c.receiveExactly(len(CONFIRMATION))
	if err != nil {
		return err
	}

	if string(data) != CONFIRMATION {
		return errors.New("confirmation failed, unexpected response")
	}

	return nil
}

func (c *Client) SendBatch(bets []Bet) error {
	serialized_data, err := SerializeBatch(bets)
	if err != nil {
		return err
	}

	batch_length := uint32(len(serialized_data))
	batch_length_bytes := make([]byte, binary.Size(batch_length))
	binary.BigEndian.PutUint32(batch_length_bytes, batch_length)
	serialized_data = append(batch_length_bytes, serialized_data...)

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

func (c *Client) receiveExactly(length int) ([]byte, error) {
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
