package common

import (
	"bufio"
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
	config ClientConfig
	conn   net.Conn
	done   chan bool
}

func (c *Client) sigtermHandler(sigterms chan os.Signal) {
	<-sigterms
	log.Infof("action: graceful_shutdown | result: in_progress | client_id: %v",
		c.config.ID,
	)

	c.done <- true

	log.Infof("action: close_socket | result: in_progress | client_id: %v",
		c.config.ID,
	)
	c.conn.Close()
	log.Infof("action: close_socket | result: success | client_id: %v",
		c.config.ID,
	)
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
		done:   make(chan bool, 1),
	}

	sigterms := make(chan os.Signal, 1)
	signal.Notify(sigterms, syscall.SIGTERM)
	go client.sigtermHandler(sigterms)

	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Fatalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// autoincremental msgID to identify every message sent
	msgID := 1

loop:
	// Send messages if the loopLapse threshold has not been surpassed
	for timeout := time.After(c.config.LoopLapse); ; {
		select {
		case <-timeout:
			log.Infof("action: timeout_detected | result: success | client_id: %v",
				c.config.ID,
			)
			break loop
		case <-c.done:
			log.Infof("action: graceful_shutdown | result: success | client_id: %v",
				c.config.ID,
			)
			break loop
		default:
		}

		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()

		// TODO: Modify the send to avoid short-write
		fmt.Fprintf(
			c.conn,
			"[CLIENT %v] Message N°%v\n",
			c.config.ID,
			msgID,
		)
		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		msgID++
		c.conn.Close()

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}
		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		select {
		case <-c.done:
			log.Infof("action: graceful_shutdown | result: success | client_id: %v",
				c.config.ID,
			)
			break loop
		case <-time.After(c.config.LoopPeriod):
		}
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

func (c *Client) ProcessBet(bet Bet) {
	c.createClientSocket()
	defer c.conn.Close()

	msg, send_err := c.sendBet(bet)
	if send_err != nil {
		log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v",
			c.config.ID,
			send_err,
		)
		return
	}

	log.Infof("action: send_bet | result: success | client_id: %v | msg: %v",
		c.config.ID,
		msg,
	)

	if receive_err := c.receiveConfirmation(); receive_err != nil {
		log.Errorf("action: receive_confirmation | result: fail | client_id: %v | error: %v",
			c.config.ID,
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
	if err := c.send(length_bytes); err != nil {
		return "", err
	}

	msg_bytes := []byte(msg)
	if err := c.send(msg_bytes); err != nil {
		return "", err
	}

	return msg, nil
}

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
