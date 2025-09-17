package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beevik/ntp"
)

// getNTPTime возвращает точное время с NTP-сервера
func getNTPTime() (time.Time, error) {
	ntpTime, err := ntp.Time("pool.ntp.org")
	if err != nil {
		return time.Time{}, fmt.Errorf("ошибка получения NTP времени: %w", err)
	}
	return ntpTime, nil
}

func main() {
	exactTime, err := getNTPTime()
	if err != nil {
		log.Printf("Ошибка: %v", err)
		os.Exit(1)
	}

	fmt.Printf("Точное время (NTP): %s\n", exactTime.Format(time.RFC3339))
	
	localTime := time.Now()
	fmt.Printf("Локальное время: %s\n", localTime.Format(time.RFC3339))
	fmt.Printf("Расхождение: %v\n", exactTime.Sub(localTime))
}
