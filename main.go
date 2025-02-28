package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/ipoluianov/diode/an"
	"github.com/ipoluianov/diode/bybit"
	"github.com/ipoluianov/diode/chart"
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

func CalculateLeastSquares(x []float64, y []float64) (float64, float64) {
	count := len(x)
	sumx := 0.0
	sumy := 0.0
	sumx2 := 0.0
	sumxy := 0.0

	for i := 0; i < count; i++ {
		sumx += x[i]
		sumy += y[i]
		sumx2 += x[i] * x[i]
		sumxy += x[i] * y[i]
	}

	a := 0.0
	b := 0.0
	if count > 2 {
		a = (float64(count)*sumxy - (sumx * sumy)) / (float64(count)*sumx2 - sumx*sumx)
		b = (sumy - a*sumx) / float64(count)
	}

	return a, b

	/*
		void LinearOLS::fill()
		    {

		        unsigned count = x_.size();
		        double sumx = 0;
		        double sumy = 0;
		        double sumx2 = 0;
		        double sumxy = 0;

		        for (unsigned i = 0; i < count; i++)
		        {
		            sumx += x_[i];
		            sumy += y_[i];
		            sumx2 += x_[i] * x_[i];
		            sumxy += x_[i] * y_[i];
		        }

		        a_ = 0;
		        b_ = 0;
		        valid_ = false;
		        if (count > 2)
		        {
		            a_ = (count * sumxy - (sumx * sumy)) / (count * sumx2 - sumx * sumx);
		            b_ = (sumy - a_ * sumx) / count;
		            valid_ = true;
		            filled_ = true;
		        }
		    }

	*/
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

func Emulate(short int, long int, stopLoss float64, takeProfit float64, p0 float64) float64 {
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
		//openPrice, _ := strconv.ParseFloat(c.OpenPrice, 64)
		closePrice, _ := strconv.ParseFloat(c.ClosePrice, 64)
		//avgPrice := (openPrice + closePrice) / 2
		prices[i] = closePrice
	}

	s1, _ := an.CalculateMACD(prices, short, long, 9)

	s1Mid := 0.0
	for _, v := range s1 {
		if v < 0 {
			s1Mid += v
		}
	}
	s1Mid /= float64(len(s1))

	posIsActive := false
	posBuyPrice := 0.0
	posOpenUSDT := 0.0
	posOpenIndex := 0

	countOfOperations := 0

	lastUSDTBalance := balanceUSDT

	waitingForNegative := false

	type Transaction struct {
		OpenIndex  int
		CloseIndex int
		Positive   bool
	}

	trs := make([]Transaction, 0)
	trends := make([]float64, len(candles))

	trendUp := false

	for i := 10; i < len(candles)-10; i++ {

		if trendUp {
			trends[i] = 1
		} else {
			trends[i] = 0
		}

		candle := candles[i]
		line := ""
		line += candle.StartTime.Format("2006-01-02 15:04:05")
		line += "\t"
		line += fmt.Sprint(prices[i])
		line += "\t"
		line += fmt.Sprint(s1[i])
		line += "\t"

		{
			// отпределяем тренд по последним 100 точкам алгоритмом наименьших квадратов
			// если коэффициент наклона прямой положительный - тренд вверх
			// если отрицательный - тренд вниз
			trendUp = false
			pointsCount := len(prices) / 10
			if i > pointsCount {
				x := make([]float64, pointsCount)
				y := make([]float64, pointsCount)
				for j := 0; j < pointsCount; j++ {
					x[j] = float64(j)
					y[j] = prices[i-j]
				}
				k, m := CalculateLeastSquares(x, y)
				_ = k
				_ = m
				if k < 0 {
					trendUp = true
				} else {
					trendUp = false
				}
			}
		}

		if posIsActive {
			currentS1LessThenPreviousS1 := s1[i] < s1[i-1]
			profitIndicator := 0.0
			{
				posSellPrice := prices[i]
				closeUSDT := balanceDEEP*posSellPrice - (posOpenUSDT * 0.002)
				profitIndicator = (closeUSDT - posOpenUSDT) / posOpenUSDT * 100.0
			}

			needClose := profitIndicator < stopLoss || profitIndicator > takeProfit || currentS1LessThenPreviousS1

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
				logger.Println(posSellPrice, "-", posBuyPrice, "=", profitInProcents, "%")
				line += "CLOSED"
				line += "\t"
				line += fmt.Sprint(profitInProcents)
				line += "%"
				line += "\t"
				waitingForNegative = true

				tr := Transaction{
					OpenIndex:  posOpenIndex,
					CloseIndex: i}

				if usdtDiff > 0 {
					tr.Positive = true
				}

				trs = append(trs, tr)
			}
		} else {
			// decision to open position - if last 2 values of s1 are positive
			if !waitingForNegative {
				needOpen := false

				if s1[i] < s1Mid+s1Mid*1 {
					if s1[i] > s1[i-1] {
						needOpen = true
					}
				}

				/*if !trendUp {
					needOpen = false
				}*/

				if needOpen {
					//if s1[i] > s1[i-1] && s1[i-1] > s1[i-2] && s1[i-2] > s1[i-3] && s1[i] < 0 {
					//logger.Println("Buy at ", prices[i])
					// Buy
					balanceDEEP += balanceUSDT / prices[i]
					posOpenUSDT = balanceUSDT
					balanceUSDT = 0
					posIsActive = true
					posBuyPrice = prices[i]
					posOpenIndex = i

					line += "OPENED"
					line += "\t"
				}
			}
		}

		if waitingForNegative {
			if s1[i] < 0 {
				waitingForNegative = false
			}
		}

		line += "\r\n"
		result += line
	}

	totalProfitInProcents := (lastUSDTBalance - initialBalanceUSDT) / initialBalanceUSDT * 100

	logger.Println("balanceUSDT:", lastUSDTBalance)
	logger.Println("totalProfitInProcents:", totalProfitInProcents, "%")
	logger.Println("countOfOperations:", countOfOperations)
	logger.Println("Commisions:", float64(countOfOperations)*(lastUSDTBalance*0.001), "USDT")

	//return totalProfitInProcents

	os.WriteFile("result.txt", []byte(result), 0644)

	{
		ch1 := chart.NewChart()
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

		img1 := ch1.DrawTrace()

		ch2 := chart.NewChart()
		ch2.SetData(s1)

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
		}

		img2 := ch2.DrawTrace()

		ch3 := chart.NewChart()
		ch3.SetData(trends)
		img3 := ch3.DrawTrace()

		ch4 := chart.NewChart()
		{
			data := make([]float64, len(prices))
			r := len(prices) / 10
			for i := 0; i < len(prices); i++ {
				data[i] = prices[i]
				if i > r {
					for j := i - r; j < i; j++ {
						data[i] += prices[j]
					}
					data[i] /= float64(r)
				}
			}

			//savitzkyGolay5(data)

			ch4.SetData(data)
		}
		img4 := ch4.DrawTrace()

		img := chart.CombineImages(img1, img2, img3, img4)

		f, _ := os.Create("02_RESULT.png")
		png.Encode(f, img)
		f.Close()
	}

	return 0
}

func main() {
	Emulate(12, 26, -0.5, 2, 0)
	/*for stopLoss := 0.5; stopLoss < 1.0; stopLoss += 0.1 {
		logger.Println("stopLoss:", stopLoss)
		for takeProfit := 0.2; takeProfit < 5.0; takeProfit += 0.1 {
			logger.Println("takeProfit:", takeProfit)
			for short := 1; short < 20; short++ {
				for long := short + 3; long < 30; long++ {
					profit := Emulate(short, long, -stopLoss, takeProfit, 0)
					if profit > 0 {
						logger.Println("-----------------------", "short:", short, "long:", long, "profit:", profit)
					}
				}
			}
		}
	}*/
	//FetchInstruments()
	//FetchData()
}
