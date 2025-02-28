package main

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/ipoluianov/diode/an"
	"github.com/ipoluianov/diode/bybit"
	"github.com/ipoluianov/gomisc/logger"
)

func FetchData() {
	dtBegin := time.Now().Add(-12 * time.Hour)
	dtEnd := time.Now()
	candles := bybit.GetCandles("DEEPUSDT", dtBegin, dtEnd, "1")
	candlesBS, _ := json.MarshalIndent(candles, "", "  ")
	os.WriteFile("candles.json", candlesBS, 0644)
}

func FetchInstruments() {
	instruments := bybit.FetchInstruments()
	instrumentsBS, _ := json.MarshalIndent(instruments, "", "  ")
	os.WriteFile("instruments.json", instrumentsBS, 0644)
}

func Analize() {
	candlesBS, _ := os.ReadFile("candles.json")
	var candles []bybit.Candle
	json.Unmarshal(candlesBS, &candles)

	prices := make([]float64, len(candles))
	for i, c := range candles {
		openPrice, _ := strconv.ParseFloat(c.OpenPrice, 64)
		closePrice, _ := strconv.ParseFloat(c.ClosePrice, 64)
		avgPrice := (openPrice + closePrice) / 2
		prices[i] = avgPrice
	}

	s1, s2 := an.CalculateMACD(prices, 12, 26, 9)
	s1BS, _ := json.MarshalIndent(s1, "", "  ")
	s2BS, _ := json.MarshalIndent(s2, "", "  ")
	os.WriteFile("s1.json", s1BS, 0644)
	os.WriteFile("s2.json", s2BS, 0644)
}

func Emulate() {
	logger.Println("Emulate begin")
	initialBalanceUSDT := 1000.0
	balanceUSDT := 1000.0
	balanceDEEP := 0.0

	candlesBS, _ := os.ReadFile("candles.json")
	var candles []bybit.Candle
	json.Unmarshal(candlesBS, &candles)

	prices := make([]float64, len(candles))
	for i, c := range candles {
		openPrice, _ := strconv.ParseFloat(c.OpenPrice, 64)
		closePrice, _ := strconv.ParseFloat(c.ClosePrice, 64)
		avgPrice := (openPrice + closePrice) / 2
		prices[i] = avgPrice
	}

	s1, _ := an.CalculateMACD(prices, 12, 26, 9)

	posIsActive := false
	posBuyPrice := 0.0

	lastUSDTBalance := balanceUSDT

	for i := 1; i < len(candles)-1; i++ {
		if posIsActive {
			currentS1LessThenPreviousS1 := s1[i] < s1[i-1]
			if currentS1LessThenPreviousS1 {
				posSellPrice := prices[i]
				profitInProcents := (posSellPrice - posBuyPrice) / posBuyPrice * 100
				diff := posSellPrice - posBuyPrice
				logger.Println(posSellPrice, "-", posBuyPrice, "=", diff, "(", profitInProcents, "%)")
				balanceUSDT += balanceDEEP * posSellPrice
				balanceDEEP = 0
				posIsActive = false
				lastUSDTBalance = balanceUSDT
			}
		} else {
			// decision to open position - if last 2 values of s1 are positive
			if i > 1 && s1[i-1] > 0 && s1[i] > 0 {
				//logger.Println("Buy at ", prices[i])
				// Buy
				balanceDEEP += balanceUSDT / prices[i]
				balanceUSDT = 0
				posIsActive = true
				posBuyPrice = prices[i]
			}
		}
	}

	totalProfitInProcents := (lastUSDTBalance - initialBalanceUSDT) / initialBalanceUSDT * 100

	logger.Println("balanceUSDT:", lastUSDTBalance)
	logger.Println("totalProfitInProcents:", totalProfitInProcents, "%")
}

func main() {
	Emulate()
	//FetchInstruments()
	//FetchData()
}
