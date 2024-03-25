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
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

loop:
	// Send messages if the loopLapse threshold has not been surpassed
	for timeout := time.After(c.config.LoopLapse); ; {
		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()
		
		// TODO: Modify the send to avoid short-write
		fmt.Fprintf(
			c.conn,
			"[CLIENT %v] Message N°%v\n",
			c.config.ID,
			msgID,
		)
		recv_chan := make(chan string)
		go recv_line(c.conn, c.config.ID, recv_chan)
		
		loop_period_chan := time.After(c.config.LoopPeriod)
		select {
			case <-timeout:
				log.Infof("action: timeout_detected | result: success | client_id: %v",
				c.config.ID,
				)
				break loop
			case <- sigs:
				break loop
			case received:= <- recv_chan:
				if received == "\n"{
					break loop
				}
				msgID++
				select{
					case <- sigs:
						break loop
					case <- loop_period_chan:
				}
			}
		c.conn.Close()
	}
	c.conn.Close()
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

//Attempts to receive a characters until one is \n from conn, if successfull sends the string through the channel.
//On failure sends "\n"
func recv_line(conn net.Conn, cli_id string, channel chan<- string){
	msg, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil {
		log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			cli_id,
			err,
		)
		channel <- "\n"
	}else{
		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			cli_id,
			msg,
		)
		channel <- msg
	}
}