package main

import (
	"encoding/csv"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

var cmd = &cobra.Command{
	Use:  "finance-prices [flags]",
	RunE: Run,
}

var output string
var timePeriod string

func main() {
	flags := cmd.PersistentFlags()
	flags.StringVarP(&output, "output", "o", "hledger", "format the output: csv,hledger,beancount")
	flags.StringVarP(&timePeriod, "time-period", "p", "1D", "time Period: 1D,5D,3M,6M,YTD,1Y,5Y,2021-10-10-2022-10-10")

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func Run(cmd *cobra.Command, args []string) error {
	var from int64
	var to int64
	now := time.Now()
	switch timePeriod {
	case "1D":
		from = now.AddDate(0, 0, -1).Unix()
		to = now.Unix()
	case "5D":
		from = now.AddDate(0, 0, -5).Unix()
		to = now.Unix()
	case "3M":
		from = now.AddDate(0, -3, 0).Unix()
		to = now.Unix()
	case "6M":
		from = now.AddDate(0, -6, 0).Unix()
		to = now.Unix()
	case "YTD":
		from = now.AddDate(0, 0, -now.YearDay()).Unix()
		to = now.Unix()
	case "1Y":
		from = now.AddDate(-1, 0, 0).Unix()
		to = now.Unix()
	case "5Y":
		from = now.AddDate(-5, 0, 0).Unix()
		to = now.Unix()
	}

	var stocks []Stock
	for _, arg := range args {
		code := arg
		switch {
		case strings.HasSuffix(arg, "SH"):
			code = strings.TrimSuffix(arg, "SH")
			code += "SS"
		}
		s, err := DownloadStockCSV(code, from, to)
		if err != nil {
			return err
		}
		reader := csv.NewReader(strings.NewReader(s))
		fields, err := reader.Read()
		if err != nil {
			return err
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

		for {
			vals, err := reader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			date := vals[dateIndex]
			price := vals[priceIndex]
			t, err := time.Parse("2006-01-02", date)
			if err != nil {
				return err
			}
			p, err := strconv.ParseFloat(price, 64)
			if err != nil {
				return err
			}
			stock := Stock{
				Symbol: arg,
				Date:   t,
				Price:  p,
			}
			stocks = append(stocks, stock)
		}
	}

	sort.Slice(stocks, func(i, j int) bool {
		if stocks[i].Date.Before(stocks[j].Date) {
			return true
		}
		return stocks[i].Date.Equal(stocks[j].Date) && stocks[i].Symbol > stocks[j].Symbol
	})
	w := &tabwriter.Writer{}
	w.Init(os.Stdout, 4, 4, 0, '\t', 0)
	defer w.Flush()
	for _, stock := range stocks {
		fmt.Fprintf(w, "P\t%s\t\"%s\"\t%.2f CNY\n", stock.Date.Format("2006-01-02"), stock.Symbol, stock.Price)
	}
	return nil
}

type Stock struct {
	Symbol string
	Date   time.Time
	Price  float64
}

// DownloadStockCSV ...
func DownloadStockCSV(code string, from, to int64) (string, error) {
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
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
