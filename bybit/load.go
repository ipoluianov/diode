package bybit

/*func Load() {
	p := logger.CurrentExePath() + "/data/bybit/ETHUSD/"
	t1 := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	t2 := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC).Unix()

	for ; t1 < t2; t1 += 86400 {
		filename := p + time.Unix(t1, 0).Format("2006-01-02") + ".txt"
		LoadData("ETHUSD", time.Unix(t1, 0), filename)
	}
}*/

// https://api-testnet.bybit.com/v5/market/kline?category=inverse&symbol=BTCUSD&interval=60&start=1670601600000&end=1670608800000
