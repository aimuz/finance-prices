package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func init() {
	Register("eastMoney", &EastMoney{})
}

type EastMoney struct{}

func (e EastMoney) Match(symbol string) bool {
	return strings.HasSuffix(symbol, ".JJ")
}

var _eastMoneyReg = regexp.MustCompile(`var Data_netWorthTrend = (.+?);`)

func (e EastMoney) Prices(symbol string, from, to int64) ([]Stock, error) {
	code := strings.TrimSuffix(symbol, ".JJ")
	url := fmt.Sprintf("https://fund.eastmoney.com/pingzhongdata/%s.js?v=%s", code, time.Now().Format("20060102150405"))
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bs := _eastMoneyReg.FindSubmatch(b)
	if len(bs) < 2 {
		return nil, nil
	}
	type t struct {
		X int64   `json:"x"`
		Y float64 `json:"y"`
	}
	var ts []t
	err = json.Unmarshal(bs[1], &ts)
	if err != nil {
		return nil, err
	}
	stocks := make([]Stock, 0, len(ts))
	for _, t2 := range ts {
		sec := t2.X / 1000
		if sec < from {
			continue
		}
		if sec > to {
			continue
		}
		t := time.Unix(sec, 0)
		stocks = append(stocks, Stock{
			Symbol: symbol,
			Date:   t,
			Price:  t2.Y,
		})
	}
	return stocks, nil
}
