package common

import (
	//"bufio"
	"net"
	"time"
	"os"
    "os/signal"
    "syscall"
	"encoding/binary"
	"fmt"
	"strconv"
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

/*	Attempts to send all bets in less than LoopLapse time, sending a batch
	every LoopPeriod seconds. If all bets are sent it returns true, athorwise it 
	returns false
*/ 
func (c *Client) send_all_bets(sigs chan os.Signal) bool{
	ack_chan := make(chan bool)
	bets_file := "/data/agency-" + c.config.ID + ".csv"
	gen := BetBatchGeneratorFrom(bets_file, c.config.MaxBatchSize)
	defer gen.Close()

	for timeout := time.After(c.config.LoopLapse); ; {
		batch, err := gen.NextBatch()
		if err != nil{
			log.Errorf("action: creando apuesta | result: fail  %v", err)
			return false
		}
		
		err = send_all(c.conn, batch.ToBytes())
		if err != nil {
			log.Errorf("action: enviando apuesta | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return false
		}

		if batch.IsEmpty(){
			log.Infof("action: enviado todas las apuestas | result: success |client_id: %v | error: %v",
			c.config.ID,
			err,
			)
			break
		}

		go recv_bet_batch_ack(c.conn, c.config.ID,ack_chan)

		loop_period_chan := time.After(c.config.LoopPeriod)

		select {
		case <-timeout:
			log.Infof("action: timeout_detected | result: success | client_id: %v",
			c.config.ID,
			)
			return false
		case <-sigs:
			return false
		case received:= <- ack_chan:
			if !received{
				return false
			}
			<- loop_period_chan
		}
	}
	return true
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	c.createClientSocket()
	defer c.conn.Close()

	if !c.send_all_bets(sigs){
		return
	}
	
	c.get_winners(sigs)
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}


func (c *Client) get_winners(sigs chan os.Signal){
	id, err := strconv.ParseUint(c.config.ID, 10, 16)
    if err != nil {
        fmt.Println("Error al convertir el string:", err)
        return
    } 

	id_bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(id_bytes, uint16(id))

	err = send_all(c.conn, id_bytes)
	if err != nil {
		log.Errorf("action: Consulta_ganadores | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return
	}

	amount_of_winners := c.recv_amount_of_winners(sigs)
	c.recv_winners(sigs, amount_of_winners)
}

// Returns the ammount_of_winners or -1 on failure
func (c *Client) recv_amount_of_winners(sigs chan os.Signal) int{
	recv_chan := make(chan int)
	go recv_positive_int(c.conn, c.config.ID ,recv_chan)
	select {
		case <-sigs:
			return -1
		case amount_of_winners:= <- recv_chan:
			if amount_of_winners > 0{
				log.Infof("action: consulta_ganadores | result: success | client_id %v | cant_ganadores: %v",
				c.config.ID,
				amount_of_winners,
				)
			}
			return amount_of_winners
	}
}

//Receives all winners dnis and logs them
func (c *Client) recv_winners(sigs chan os.Signal, amount_of_winners int){
	recv_chan := make(chan int)
	for i:=0; i<amount_of_winners; i++{
		go recv_positive_int(c.conn, c.config.ID ,recv_chan)
		select {
			case <-sigs:
				return 
			case winner:= <- recv_chan:
				if winner < 0{
					return
				}
				log.Infof("action: Recibir ganador | result: success | client_id: %v | winner: %v",
					c.config.ID,
					winner,
				)
		}
	}
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

//Attempts to receive a positive int from conn, if successfull sends it through the channel.
//On failure sends -1
func recv_positive_int(conn net.Conn, cli_id string, channel chan<- int){
	const INT_AMOUNT_OF_BYTES = 4 
	int_bytes, err := recv_exactly(conn, INT_AMOUNT_OF_BYTES)

	if err != nil {
		log.Errorf("action: recibiendo ganadores | result: fail | client_id: %v | error: %v",
			cli_id,
			err,
		)
		channel <- int(-1)
	}else{
		channel <- int(binary.BigEndian.Uint32(int_bytes))
	}
}
