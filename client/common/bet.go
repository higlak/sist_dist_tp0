package common

import (
	"fmt"
	"strconv"
	"time"
	"os"
	"encoding/binary"
	"bufio"
	"strings"
	"io"
	//log "github.com/sirupsen/logrus"
)

type Bet struct {
	agency		  uint16
	birth_date    time.Time
	dni           uint32
	lotteryNumber uint32
	name          string
	lastName      string
}

//NOMBRE=Santiago Lionel, APELLIDO=Lorca, DOCUMENTO=30904465, NACIMIENTO=1999-03-17 y NUMERO=7574

func BetFromEnv() *Bet{
	return BetFromStrings(
		os.Getenv("NACIMIENTO"),
		os.Getenv("DOCUMENTO"),
		os.Getenv("NUMERO"),
		os.Getenv("NOMBRE"),
		os.Getenv("APELLIDO"),
	)
}

func BetFromStrings(birth_date_str string, dni_str string, number_str string, name string, last_name string) *Bet{
	const SAMPLE_DATE = "2006-01-02"
	
	agency, err := strconv.ParseUint(os.Getenv("CLI_ID"), 10, 16)
    if err != nil {
        fmt.Println("Error al convertir el string:", err)
        return nil
    } 

	birth_date, err := time.Parse(SAMPLE_DATE, birth_date_str)
    if err != nil {
        fmt.Println("Error al analizar la fecha:", err)
        return nil
    }

	dni, err := strconv.ParseUint(dni_str, 10, 32)
    if err != nil {
        fmt.Println("Error al convertir el string:", err)
        return nil
    } 

	lottery_number, err := strconv.ParseUint(number_str, 10, 32)
    if err != nil {
        fmt.Println("Error al convertir el string:", err)
        return nil
    }

	bet := &Bet{
		agency: uint16(agency),
		birth_date: birth_date,
		dni: uint32(dni),
		lotteryNumber: uint32(lottery_number),
		name: name,
		lastName: last_name,
	}
	return bet
}

func dateToBytes(date time.Time) []byte {
	var data []byte

	data = append(data, byte(date.Day()))
	data = append(data, byte(date.Month()))
	year_bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(year_bytes, uint16(date.Year()))
	data = append(data, year_bytes...)
	return data
}

func (bet *Bet) ToBytes() []byte {
	const MAX_NAME_LEN = 127
	var data []byte 
	
	agency_bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(agency_bytes, bet.agency)
	data = append(data, agency_bytes...)
	
	data = append(data,dateToBytes(bet.birth_date)...)

	dni_bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(dni_bytes, bet.dni)
	data = append(data, dni_bytes...)

	lottery_number_bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lottery_number_bytes, bet.lotteryNumber)
	data = append(data, lottery_number_bytes...)

	full_name := truncateString(bet.name, MAX_NAME_LEN) + ";" + truncateString(bet.lastName, MAX_NAME_LEN)
	data = append(data, byte(len(full_name)))
	data = append(data, []byte(full_name)...)
	return data
}

type BetBatch struct{
	bets []Bet
}

func (batch *BetBatch) ToBytes() []byte {
	var data []byte
	data = append(data, byte(len(batch.bets)))
	for _, bet := range batch.bets{
		data = append(data, bet.ToBytes()...)
	}
	return data
}

func (batch *BetBatch) addBet(bet Bet) {
	batch.bets = append(batch.bets, bet)
}

type BetBatchGenerator struct{
	file *os.File
	reader *bufio.Reader
	batchSize byte
}

func BetBatchGeneratorFrom(path string, batch_size byte) *BetBatchGenerator{
	file, err := os.Open(path)
    if err != nil {
        fmt.Println("Error al abrir el archivo:", err)
		file.Close()
        return nil
    }
	
	return &BetBatchGenerator{
		file: file,
		reader: bufio.NewReader(file),
		batchSize: batch_size,
	}
}

func (gen *BetBatchGenerator) NextBatch() (*BetBatch, error){
	var batch BetBatch
	for i := 0; i < int(gen.batchSize); i++ {
		bet, err := gen.NextBet()
		if err != nil{
			if err != io.EOF{
				return nil, err
			}
			break
		}
		//la entrada en el csv era valida
		if bet != nil{
			batch.addBet(*bet)
		}
	}
	return &batch, nil
}

func (gen *BetBatchGenerator) NextBet() (*Bet, error){
	const BIRTH_DATE_POSITION = 3
	const DNI_POSITION = 2
	const NUMBER_POSITION = 4
	const NAME_POSITION = 0
	const LAST_NAME_POSITION = 1

	line, err := gen.reader.ReadString('\n')
	if err != nil{
		return nil, err
	}
	fields := strings.Split(strings.TrimRight(line, "\r\n\000 "), ",")
	bet := BetFromStrings(
		fields[BIRTH_DATE_POSITION], 
		fields[DNI_POSITION], 
		fields[NUMBER_POSITION], 
		fields[NAME_POSITION], 
		fields[LAST_NAME_POSITION])
	return bet, nil
}


func (batch *BetBatchGenerator) Close() {
	batch.file.Close()
}

func truncateString(str string, maxLength int) string {
	if len(str) > maxLength {
		return str[:maxLength]
	}
	return str
}