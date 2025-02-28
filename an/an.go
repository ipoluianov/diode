package an

func calculateEMA(prices []float64, period int) []float64 {
	if period <= 0 || len(prices) < period {
		return nil
	}

	multiplier := 2.0 / float64(period+1)
	ema := make([]float64, len(prices))

	// Начальное значение EMA — это простая скользящая средняя первых period значений
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	// Вычисляем оставшиеся значения EMA
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i]-ema[i-1])*multiplier + ema[i-1]
	}

	return ema
}

// calculateMACD рассчитывает MACD и сигнальную линию
func CalculateMACD(prices []float64, shortPeriod, longPeriod, signalPeriod int) ([]float64, []float64) {
	if shortPeriod >= longPeriod || len(prices) < longPeriod {
		return nil, nil
	}

	// Рассчитываем короткую и длинную EMA
	shortEMA := calculateEMA(prices, shortPeriod)
	longEMA := calculateEMA(prices, longPeriod)

	if shortEMA == nil || longEMA == nil {
		return nil, nil
	}

	// Вычисляем MACD (разницу между короткой и длинной EMA)
	macd := make([]float64, len(prices))
	for i := longPeriod - 1; i < len(prices); i++ {
		macd[i] = shortEMA[i] - longEMA[i]
	}

	// Рассчитываем сигнальную линию (EMA от MACD)
	signalLine := calculateEMA(macd[longPeriod-1:], signalPeriod)

	return macd, signalLine
}
