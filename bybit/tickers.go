package bybit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ipoluianov/gomisc/logger"
)

// https://api-testnet.bybit.com/v5/market/tickers?category=inverse&symbol=BTCUSD

func FetchInstruments() []string {
	logger.Println("bybit FetchInstruments begin")
	time.Sleep(100 * time.Millisecond)
	requestLine := "https://api.bybit.com/v5/market/instruments-info?category=spot"
	resp, err := http.Get(requestLine)
	if err != nil {
		logger.Println("bybit FetchInstruments error:", err)
		return nil
	}
	logger.Println("bybit FetchInstruments stutus:", resp.StatusCode)
	buf := make([]byte, 10*1024*1024)
	data := make([]byte, 0)
	for {
		n, err := resp.Body.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			fmt.Println("err:", err)
		}
		data = append(data, buf[:n]...)
	}
	logger.Println("bybit FetchInstruments data size:", len(data))

	var v GetInstrumentsResponse
	err = json.Unmarshal(data, &v)
	if err != nil {
		fmt.Println("Unmarshal error:", err)
	}
	result := make([]string, 0)

	for _, symbol := range v.Result.List {
		result = append(result, symbol.Symbol)
	}

	logger.Println("bybit FetchInstruments end")
	return result
}

func UpdateInstruments() {
	logger.Println("UpdateInstruments begin")
	filePath := logger.CurrentExePath() + "/data/bybit/instruments.txt"
	instruments := FetchInstruments()
	l := GetInstruments()
	logger.Println("UpdateInstruments existing count:", len(l))
	logger.Println("UpdateInstruments fatched  count:", len(instruments))
	m := make(map[string]struct{})
	for _, li := range l {
		m[li] = struct{}{}
	}
	for _, li := range instruments {
		m[li] = struct{}{}
	}
	unsortedList := make([]string, 0)
	for mi := range m {
		unsortedList = append(unsortedList, mi)
	}
	sort.Slice(unsortedList, func(i, j int) bool {
		return unsortedList[i] < unsortedList[j]
	})
	sortedList := unsortedList
	fileContent := ""
	for _, li := range sortedList {
		fileContent += li
		fileContent += "\r\n"
	}

	dirName := filepath.Dir(filePath)
	os.MkdirAll(dirName, 0777)
	err := os.WriteFile(filePath, []byte(fileContent), 0666)
	if err != nil {
		logger.Println("UpdateInstruments write file error:", err)
	}
	logger.Println("UpdateInstruments end")
}

func ThUpdateInstruments() {
	for !stopping {
		UpdateInstruments()

		for i := 0; i < 60; i++ {
			time.Sleep(time.Second)
			if stopping {
				break
			}
		}
		if stopping {
			break
		}
	}
}

func GetInstruments() []string {
	filePath := logger.CurrentExePath() + "/data/bybit/instruments.txt"
	bs, err := os.ReadFile(filePath)
	l := strings.FieldsFunc(string(bs), func(r rune) bool {
		return r == '\r' || r == '\n'
	})
	if err != nil {
		logger.Println("UpdateInstruments read file error:", err)
	}
	m := make(map[string]struct{})
	for _, li := range l {
		m[li] = struct{}{}
	}
	unsortedList := make([]string, 0)
	for mi := range m {
		unsortedList = append(unsortedList, mi)
	}
	sort.Slice(unsortedList, func(i, j int) bool {
		return unsortedList[i] < unsortedList[j]
	})
	sortedList := unsortedList
	return sortedList
}
