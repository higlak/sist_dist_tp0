package common

import (
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

loop:
	// Send messages if the loopLapse threshold has not been surpassed
	for timeout := time.After(c.config.LoopLapse); ; {
		c.createClientSocket()

		bet := BetFromEnv()
		if bet == nil{
			log.Errorf("action: creando apuesta | result: fail | variables de entorno no inicializadas")
			c.conn.Close()
			return
		}
		err := send_all(c.conn, bet.ToBytes())
		
		if err != nil {
        	log.Errorf("action: enviando apuesta | result: fail | client_id: %v | error: %v",
                c.config.ID,
				err,
			)
			c.conn.Close()
			return 
		}

		ack_chan := make(chan bool)
		go recv_bet_ack(c.conn, c.config.ID,ack_chan)
		loop_period_chan := time.After(c.config.LoopPeriod)
				
		select {
			case <-timeout:
				log.Infof("action: timeout_detected | result: success | client_id: %v",
				c.config.ID,
				)
				break loop
			case <- sigs:
				break loop
			case ack:= <- ack_chan:
				if !ack{
					break loop
				}
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

//Attempts to receive a byte from conn, if successfull sends true through the channel.
//On failure sends false
func recv_bet_ack(conn net.Conn, cli_id string, channel chan<- bool){
	const ANSWEAR_BYTES = 1
	_,err := recv_exactly(conn, ANSWEAR_BYTES)

	if err != nil {
		log.Errorf("action: apuesta enviada | result: fail | client_id: %v | error: %v",
			cli_id,
			err,
		)
		channel <- false
	}else{
		log.Errorf("action: apuesta enviada | result: success | client_id: %v ",
			cli_id,)
		channel <- true
	}
}