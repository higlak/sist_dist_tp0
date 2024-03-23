package common

import (
	//"bufio"
	"net"
	"time"
	"os"
    "os/signal"
    "syscall"
	//"fmt"
	//"io"
	
	log "github.com/sirupsen/logrus"
)


// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            	string
	ServerAddress 	string
	LoopLapse     	time.Duration
	LoopPeriod    	time.Duration
	MaxBatchSize	byte
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

	bets_file := "/data/agency-" + c.config.ID + ".csv"
	gen := BetBatchGeneratorFrom(bets_file, c.config.MaxBatchSize)
	c.createClientSocket()
	defer c.conn.Close()
	defer gen.Close()

	loop:
	// Send messages if the loopLapse threshold has not been surpassed
	for timeout := time.After(c.config.LoopLapse); ; {

		batch, err := gen.NextBatch()
		if err != nil{
			log.Errorf("probleaction: creando apuesta | result: fail  %v", err)
			return
		}
		
		err = send_all(c.conn, batch.ToBytes())
		if err != nil {
			log.Errorf("action: enviando apuesta | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return 
		}

		if batch.IsEmpty(){

			log.Infof("action: enviado todas las apuestas | result: success |client_id: %v | error: %v",
			c.config.ID,
			err,
			)
			break
		}

		ack_chan := make(chan bool)
		go recv_bet_batch_ack(c.conn, c.config.ID,ack_chan)

		loop_period_chan := time.After(c.config.LoopPeriod)

		select {
		case <-timeout:
			log.Infof("action: timeout_detected | result: success | client_id: %v",
			c.config.ID,
			)
			break loop
		case <-sigs:
			break loop
		case received:= <- ack_chan:
			if !received{
				log.Infof("pase por aca:")
				break loop
			}
			<- loop_period_chan
		}
	}
	
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

//Attempts to receive a byte from conn, if successfull sends true through the channel.
//On failure sends false
func recv_bet_batch_ack(conn net.Conn, cli_id string, channel chan<- bool){
	const ANSWEAR_BYTES = 1
	_,err := recv_exactly(conn, ANSWEAR_BYTES)

	if err != nil {
		log.Errorf("action: batch_enviada | result: fail | client_id: %v | error: %v",
			cli_id,
			err,
		)
		channel <- false
	}else{
		channel <- true
	}
}