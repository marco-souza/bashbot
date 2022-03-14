package usecases

import (
	"fmt"

	"github.com/marco-souza/bashbot/pkg/services"
)

func GetSymbolCandles() {
	done := make(chan bool, 1)
	candleChan := services.WatchSymbol("btcusdt", "1m", &done)

	for {
		select {
		case candle := <-candleChan:
			fmt.Println("candle: ", candle)
		}
	}
}
