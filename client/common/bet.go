package common

import (
	"fmt"
	"strconv"
	"time"
	"os"
	"encoding/binary"
)

type Bet struct {
	agency		  uint16
	birth_date    time.Time
	dni           uint32
	lotteryNumber uint32
	name          string
	lastName      string
}

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

func truncateString(str string, maxLength int) string {
	if len(str) > maxLength {
		return str[:maxLength]
	}
	return str
}