package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func init() {
	Register("yahoo", &Yahoo{})
}

type Yahoo struct{}

func (y Yahoo) Match(symbol string) bool {
	switch {
	case strings.HasSuffix(symbol, ".SS"):
	case strings.HasSuffix(symbol, ".SH"):
	case strings.HasSuffix(symbol, ".SZ"):
	default:
		return false
	}
	return true
}

func (y Yahoo) Prices(symbol string, from, to int64) ([]Stock, error) {
	code := symbol
	if strings.HasSuffix(symbol, ".SH") {
		code = strings.TrimSuffix(symbol, ".SH") + ".SS"
	}
	period1 := strconv.FormatInt(from, 10)
	period2 := strconv.FormatInt(to, 10)
	params := url.Values{
		"period1":              []string{period1},
		"period2":              []string{period2},
		"interval":             []string{"1d"},
		"events":               []string{"history"},
		"includeAdjustedClose": []string{"true"},
	}

	// https://query1.finance.yahoo.com/v7/finance/download/000151.SZ?period1=1634204059&period2=1665740059&interval=1d&events=history&includeAdjustedClose=true
	u := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/download/%s?%s", code, params.Encode())
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	reader := csv.NewReader(resp.Body)
	fields, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var dateIndex, priceIndex int
	for index, field := range fields {
		switch field {
		case "Date":
			dateIndex = index
		case "Close":
			priceIndex = index
		}
	}

	var stocks []Stock
	var record []string
	for {
		record, err = reader.Read()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return stocks, err
		}
		date := record[dateIndex]
		price := record[priceIndex]

		stock := Stock{Symbol: symbol}

		stock.Date, err = time.Parse("2006-01-02", date)
		if err != nil {
			continue
		}
		stock.Price, err = strconv.ParseFloat(price, 64)
		if err != nil {
			continue
		}
		stocks = append(stocks, stock)
	}
}
