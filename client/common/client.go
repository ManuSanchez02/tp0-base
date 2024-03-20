package common

import (
	"bufio"
	"fmt"
	"net"
	"time"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

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

func (c *Client) sigterm_handler(sigterms chan os.Signal) {
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
		done: make(chan bool, 1),
	}

	sigterms := make(chan os.Signal, 1)
	signal.Notify(sigterms, syscall.SIGTERM)
	go client.sigterm_handler(sigterms)

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
		case <- time.After(c.config.LoopPeriod):
		}
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
