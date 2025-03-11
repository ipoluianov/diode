package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/ipoluianov/diode/an"
	"github.com/ipoluianov/diode/bybit"
	"github.com/ipoluianov/gomisc/logger"
)

func FetchData() {
	dtBegin := time.Now().Add(-12 * time.Hour)
	dtEnd := time.Now()
	candles := bybit.GetCandles("BTCUSDT", dtBegin, dtEnd, "1")
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

func savitzkyGolay5(data []float64) {
	size := len(data)
	if size < 5 {
		return // Недостаточно точек
	}

	// Коэффициенты для окна 5 точек (полином 2-й степени)
	coeffs := []float64{-3.0 / 35, 12.0 / 35, 17.0 / 35, 12.0 / 35, -3.0 / 35}

	smoothed := make([]float64, size)

	for i := 2; i < size-2; i++ {
		smoothed[i] = coeffs[0]*data[i-2] + coeffs[1]*data[i-1] +
			coeffs[2]*data[i] + coeffs[3]*data[i+1] +
			coeffs[4]*data[i+2]
	}

	// Копируем сглаженные данные обратно
	for i := 2; i < size-2; i++ {
		data[i] = smoothed[i]
	}
}

func CalculateRSI(prices []float64) float64 {
	upWards := 0.0
	downWards := 0.0

	lastPrice := prices[0]
	for i := 1; i < len(prices); i++ {
		diff := prices[i] - lastPrice
		if diff > 0 {
			upWards += diff
		} else {
			downWards += math.Abs(diff)
		}
	}

	rs := upWards / downWards
	rsi := 100 - 100/(1+rs)

	return rsi
}

func CalculateEMAPoint(prices []float64, period int) float64 {
	ema := 0.0
	for i := 0; i < period; i++ {
		ema += prices[i]
	}
	ema /= float64(period)
	return ema
}

func Emulate(short int, long int, stopLoss float64, takeProfit float64, p0 int) float64 {
	// logger.Println("Emulate begin")
	initialBalanceUSDT := 1000.0
	balanceUSDT := 1000.0
	balanceDEEP := 0.0

	result := ""

	candlesBS, _ := os.ReadFile("candles.json")
	var candles []bybit.Candle
	json.Unmarshal(candlesBS, &candles)

	sort.Slice(candles, func(i, j int) bool {
		return candles[i].StartTime.Before(candles[j].StartTime)
	})

	prices := make([]float64, len(candles))
	for i, c := range candles {
		openPrice, _ := strconv.ParseFloat(c.OpenPrice, 64)
		closePrice, _ := strconv.ParseFloat(c.ClosePrice, 64)
		avgPrice := (openPrice + closePrice) / 2
		prices[i] = avgPrice
	}

	posIsActive := false
	//posBuyPrice := 0.0
	posOpenUSDT := 0.0
	posOpenIndex := 0

	countOfOperations := 0

	lastUSDTBalance := balanceUSDT

	type Transaction struct {
		OpenIndex  int
		CloseIndex int
		Positive   bool
	}

	trs := make([]Transaction, 0)

	//rsiWindow := 60

	rsiData := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		rsiData[i] = 50
	}

	emaSlowData := make([]float64, len(prices))
	emaFastData := make([]float64, len(prices))

	skipFirstPoints := 120

	for i := skipFirstPoints; i < len(prices); i++ {
		candle := candles[i]
		line := ""
		line += candle.StartTime.Format("2006-01-02 15:04:05")
		line += "\t"
		line += fmt.Sprint(prices[i])
		line += "\t"

		rsi := CalculateRSI(prices[i-p0 : i])
		rsiData[i] = rsi

		// replace last 5 points with EMA
		//rsiEmaRange := 5
		//rsi = CalculateEMAPoint(rsiData[i-rsiEmaRange+1:i+1], rsiEmaRange)
		//rsiData[i] = rsi

		//short = 12
		//long = 50

		emaSlow := CalculateEMAPoint(prices[i-long:i], long)
		emaSlowData[i] = emaSlow

		emaFast := CalculateEMAPoint(prices[i-short:i], short)
		emaFastData[i] = emaFast

		trendUp := (emaFast - emaSlow) > (emaSlow * 0.000)

		line += "RSI:" + fmt.Sprint(math.Round(rsi))
		line += "\t"

		if posIsActive {
			//currentS1LessThenPreviousS1 := s1[i] < s1[i-1]
			profitIndicator := 0.0
			{
				posSellPrice := prices[i]
				closeUSDT := balanceDEEP*posSellPrice - (posOpenUSDT * 0.001)
				profitIndicator = (closeUSDT - posOpenUSDT) / posOpenUSDT * 100.0
			}

			needClose := profitIndicator < stopLoss || profitIndicator > takeProfit || rsi > 70

			if i-posOpenIndex < 5 {
				needClose = false
			}

			if needClose {
				posSellPrice := prices[i]
				//profitInProcents := (posSellPrice - posBuyPrice) / posBuyPrice * 100
				//diff := posSellPrice - posBuyPrice
				balanceUSDT += balanceDEEP*posSellPrice - (posOpenUSDT * 0.002)
				balanceDEEP = 0
				posIsActive = false
				lastUSDTBalance = balanceUSDT

				usdtDiff := balanceUSDT - posOpenUSDT
				profitInProcents := usdtDiff / posOpenUSDT * 100
				countOfOperations++
				//logger.Println(posSellPrice, "-", posBuyPrice, "=", profitInProcents, "%")
				line += "CLOSED"
				line += "\t"
				line += fmt.Sprint(profitInProcents)
				line += "%"
				line += "\t"

				tr := Transaction{
					OpenIndex:  posOpenIndex,
					CloseIndex: i}

				if usdtDiff > 0 {
					tr.Positive = true
				}

				trs = append(trs, tr)
			}
		} else {
			needOpen := false

			if rsi < 30 && trendUp {
				needOpen = true
			}

			if needOpen {
				balanceDEEP += balanceUSDT / prices[i]
				posOpenUSDT = balanceUSDT
				balanceUSDT = 0
				posIsActive = true
				//posBuyPrice = prices[i]
				posOpenIndex = i

				line += "OPENED"
				line += "\t"
			}
		}

		line += "\r\n"
		result += line
	}

	// Fill EMAs first points with first value
	for i := 0; i < skipFirstPoints; i++ {
		emaSlowData[i] = emaSlowData[skipFirstPoints]
		emaFastData[i] = emaFastData[skipFirstPoints]
	}

	totalProfitInProcents := (lastUSDTBalance - initialBalanceUSDT) / initialBalanceUSDT * 100

	/*logger.Println("balanceUSDT:", lastUSDTBalance)
	logger.Println("totalProfitInProcents:", totalProfitInProcents, "%")
	logger.Println("countOfOperations:", countOfOperations)
	logger.Println("Commisions:", float64(countOfOperations)*(lastUSDTBalance*0.001), "USDT")

	//return totalProfitInProcents

	os.WriteFile("result.txt", []byte(result), 0644)

	{
		// CHART 1
		ch1 := chart.NewChart()
		{
			ch1.SetText(fmt.Sprint(totalProfitInProcents))
			ch1.SetData(prices)

			{
				lines := make([]int, 0)
				for _, tr := range trs {
					lines = append(lines, tr.OpenIndex)
				}
				ch1.SetLines1(lines)

				lines = make([]int, 0)
				for _, tr := range trs {
					lines = append(lines, tr.CloseIndex)
				}
				ch1.SetLines2(lines)

				areas := make([]chart.Area, 0)
				for _, tr := range trs {
					areas = append(areas, chart.Area{Index1: tr.OpenIndex, Index2: tr.CloseIndex, Good: tr.Positive})
				}
				ch1.Areas = areas
			}
		}
		img1 := ch1.DrawTrace()

		// CHART 2
		ch2 := chart.NewChart()
		{
			ch2.SetText("RSI")
			ch2.SetData(rsiData)

			{
				lines := make([]int, 0)
				for _, tr := range trs {
					lines = append(lines, tr.OpenIndex)
				}
				ch2.SetLines1(lines)

				lines = make([]int, 0)
				for _, tr := range trs {
					lines = append(lines, tr.CloseIndex)
				}
				ch2.SetLines2(lines)

				areas := make([]chart.Area, 0)
				for _, tr := range trs {
					areas = append(areas, chart.Area{Index1: tr.OpenIndex, Index2: tr.CloseIndex, Good: tr.Positive})
				}
				ch2.Areas = areas
			}

		}
		img2 := ch2.DrawTrace()

		// CHART 3
		ch3 := chart.NewChart()
		{
			ch3.SetText("EMA")
			ch3.SetData(emaSlowData)
			ch3.SetData2(emaFastData)

			{
				lines := make([]int, 0)
				for _, tr := range trs {
					lines = append(lines, tr.OpenIndex)
				}
				ch3.SetLines1(lines)

				lines = make([]int, 0)
				for _, tr := range trs {
					lines = append(lines, tr.CloseIndex)
				}
				ch3.SetLines2(lines)

				areas := make([]chart.Area, 0)
				for _, tr := range trs {
					areas = append(areas, chart.Area{Index1: tr.OpenIndex, Index2: tr.CloseIndex, Good: tr.Positive})
				}
				ch3.Areas = areas
			}
		}
		img3 := ch3.DrawTrace()

		// CHART 4
		ch4 := chart.NewChart()
		{
		}
		img4 := ch4.DrawTrace()

		img := chart.CombineImages(img1, img2, img3, img4)

		f, _ := os.Create("02_RESULT.png")
		png.Encode(f, img)
		f.Close()
	}*/

	return totalProfitInProcents
}

func main() {
	//prof := Emulate(12, 26, -0.2, 2, 0)
	//fmt.Println("Profit:", prof)

	result := ""

	for short := 5; short < 20; short++ {
		logger.Println("short:", short)
		for long := short + 5; long < short+50; long++ {
			logger.Println("long:", long)
			for p0 := short; p0 < 110; p0++ {
				line := ""
				line += fmt.Sprint(short)
				line += "\t"
				line += fmt.Sprint(long)
				line += "\t"
				line += fmt.Sprint(p0)
				line += "\r\n"
				result += line
				profit := Emulate(short, long, -0.5, 3, p0)
				if profit > 0 {
					logger.Println("-----------------------", "short:", short, "long:", long, "profit:", profit)
				}
			}
		}
	}

	os.WriteFile("calc_result.txt", []byte(result), 0644)

	//FetchInstruments()
	//FetchData()
}
