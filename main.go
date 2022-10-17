package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"sort"
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

type PriceProvider interface {
	Match(symbol string) bool
	Prices(symbol string, from, to int64) ([]Stock, error)
}

var priceProviders = map[string]PriceProvider{}

func Register(name string, p PriceProvider) {
	priceProviders[name] = p
}

func Run(_ *cobra.Command, args []string) error {
	var from time.Time
	to := time.Now()
	switch timePeriod {
	case "1D":
		from = to.AddDate(0, 0, -1)
	case "5D":
		from = to.AddDate(0, 0, -5)
	case "3M":
		from = to.AddDate(0, -3, 0)
	case "6M":
		from = to.AddDate(0, -6, 0)
	case "YTD":
		from = time.Date(to.Year(), 1, 1, 0, 0, 0, 0, to.Location())
	case "1Y":
		from = to.AddDate(-1, 0, 0)
	case "5Y":
		from = to.AddDate(-5, 0, 0)
	}

	var stocks []Stock
	for _, arg := range args {
		code := arg
		for _, provider := range priceProviders {
			if !provider.Match(code) {
				continue
			}
			ss, err := provider.Prices(code, from.Unix(), to.Unix())
			if err != nil {
				return err
			}
			stocks = append(stocks, ss...)
		}
	}

	sort.Sort(Stocks(stocks))
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

type Stocks []Stock

func (s Stocks) Len() int {
	return len(s)
}

func (s Stocks) Less(i, j int) bool {
	if s[i].Date.Before(s[j].Date) {
		return true
	}
	return s[i].Date.Equal(s[j].Date) && s[i].Symbol > s[j].Symbol
}

func (s Stocks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
