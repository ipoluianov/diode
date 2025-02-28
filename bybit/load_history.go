package bybit

import (
	"os"
	"time"

	"github.com/ipoluianov/gomisc/logger"
)

func ThLoad() {
	for !stopping {
		LoadNext()
		for i := 0; i < 10; i++ {
			time.Sleep(10 * time.Millisecond)
			if stopping {
				break
			}
		}
		if stopping {
			break
		}
	}

}

func LoadNext() {
	minDay := int64(19720)
	maxDay := time.Now().Unix()/86400 - 1
	tickers := GetInstruments()
	for _, t := range tickers {
		for day := minDay; day < maxDay; day++ {
			if !HasData(t, day) {
				tm := TimeByDayIndex(day)
				LoadData(t, tm, GetFileNameByDate(t, tm))
				return
			}
		}
	}
}

func HasData(symbol string, dayIndex int64) bool {
	filePath := GetFileNameByDate(symbol, time.Unix(dayIndex*86400, 0))
	_, err := os.Lstat(filePath)
	return err == nil
}

func GetFileNameByDate(symbol string, t time.Time) string {
	p := logger.CurrentExePath() + "/data/bybit/" + symbol + "/"
	filename := t.Format("2006-01-02") + ".txt"
	return p + filename
}
