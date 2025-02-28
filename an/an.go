package an

func calculateEMA(prices []float64, period int, skip int) []float64 {
	result := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		if i > skip {
			sum := 0.0
			for j := i - period; j < i; j++ {
				sum += prices[j]
			}
			v := sum / float64(period)
			result[i] = v
		}
	}
	return result
}

// calculateMACD рассчитывает MACD и сигнальную линию
func CalculateMACD(prices []float64, shortPeriod, longPeriod, signalPeriod int) ([]float64, []float64) {
	if shortPeriod >= longPeriod || len(prices) < longPeriod {
		return nil, nil
	}

	// Рассчитываем короткую и длинную EMA
	shortEMA := calculateEMA(prices, shortPeriod, longPeriod)
	longEMA := calculateEMA(prices, longPeriod, longPeriod)

	if shortEMA == nil || longEMA == nil {
		return nil, nil
	}

	// Вычисляем MACD (разницу между короткой и длинной EMA)
	macd := make([]float64, len(prices))
	for i := longPeriod - 1; i < len(prices); i++ {
		macd[i] = shortEMA[i] - longEMA[i]
	}

	// Рассчитываем сигнальную линию (EMA от MACD)
	signalLine := calculateEMA(macd[longPeriod-1:], signalPeriod, longPeriod*2)

	return macd, signalLine
}
