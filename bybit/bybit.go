package bybit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/ipoluianov/gomisc/logger"
)

var stopping bool

func Start() {
	logger.Println("bybit Start")
	go ThUpdateInstruments()
	go ThLoad()
}

func Stop() {
	logger.Println("bybit Stop")
	stopping = false
}

type HeaderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
}

type StringList []string

type GetCandlesResponseInt struct {
	Symbol   string       `json:"symbol"`
	Category string       `json:"category"`
	List     []StringList `json:"list"`
}

type GetCandlesResponse struct {
	HeaderResponse
	Result GetCandlesResponseInt `json:"result"`
}

type GetInstrumentsItem struct {
	Symbol string `json:"symbol"`
}

type GetInstrumentsResponseInt struct {
	Category string               `json:"category"`
	List     []GetInstrumentsItem `json:"list"`
}

type GetInstrumentsResponse struct {
	HeaderResponse
	Result GetInstrumentsResponseInt `json:"result"`
}

type Candle struct {
	StartTime  time.Time
	OpenPrice  string
	HighPrice  string
	LowPrice   string
	ClosePrice string
	Volume     string
	Turnover   string
}

func GetCandles(symbol string, startDT time.Time, endDT time.Time, interval string) []Candle {
	time.Sleep(100 * time.Millisecond)
	start := fmt.Sprint(startDT.UnixMilli())
	end := fmt.Sprint(endDT.UnixMilli() - 1)
	requestLine := "https://api.bybit.com/v5/market/kline?category=spot&symbol=" + symbol + "&interval=" + interval + "&start=" + start + "&end=" + end + "&limit=1000"
	resp, err := http.Get(requestLine)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	fmt.Println("Status:", resp.StatusCode)
	buf := make([]byte, 10*1024*1024)
	data := make([]byte, 0)
	for {
		n, err := resp.Body.Read(buf)
		if n == 0 {
			//fmt.Println("0 received")
			break
		}
		if err != nil {
			fmt.Println("err:", err)
		}
		data = append(data, buf[:n]...)
		//buf = buf[:n]

	}
	//fmt.Println(string(data), err)

	var v GetCandlesResponse
	err = json.Unmarshal(data, &v)
	if err != nil {
		fmt.Println("Unmarshal error:", err)
	}
	result := make([]Candle, 0)
	for _, item := range v.Result.List {
		var c Candle
		timeAsIntMs, _ := strconv.ParseInt(item[0], 10, 64)
		c.StartTime = time.Unix(timeAsIntMs/1000, 0)
		c.OpenPrice = item[1]
		c.HighPrice = item[2]
		c.LowPrice = item[3]
		c.ClosePrice = item[4]
		c.Volume = item[5]
		c.Turnover = item[6]
		result = append(result, c)
	}
	return result
}

func LoadData(symbol string, date time.Time, fileName string) {
	date1 := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	date1_end := date1.Add(12 * time.Hour).Add(-1 * time.Millisecond)
	date2 := date1.Add(12 * time.Hour)
	date2_end := date2.Add(12 * time.Hour).Add(-1 * time.Millisecond)
	fmt.Println("LoadData", symbol, date1, date1_end, date2, date2_end)

	res1 := GetCandles(symbol, date1, date1_end, "1")
	res2 := GetCandles(symbol, date2, date2_end, "1")
	res := make([]Candle, 0)
	res = append(res, res1...)
	res = append(res, res2...)

	sort.Slice(res, func(i, j int) bool {
		return res[i].StartTime.UnixMilli() < res[j].StartTime.UnixMilli()
	})
	csv := ""
	for _, i := range res {
		line := i.StartTime.UTC().Format("2006-01-02 15:04:05")
		line += "\t"
		line += i.OpenPrice
		line += "\t"
		line += i.LowPrice
		line += "\t"
		line += i.HighPrice
		line += "\t"
		line += i.ClosePrice
		line += "\t"
		line += i.Volume
		line += "\t"
		csv += line + "\r\n"
	}

	fmt.Println("Loaded count:", len(res))

	dirName := filepath.Dir(fileName)

	os.MkdirAll(dirName, 0777)
	os.WriteFile(fileName, []byte(csv), 0666)
}

func ParseDate(value string) time.Time {
	t, _ := time.Parse("2006-01-02", value)
	return t
}

func TimeByDayIndex(dayIndex int64) time.Time {
	unixTime := dayIndex * 86400
	return time.Unix(int64(unixTime), 0)
}
