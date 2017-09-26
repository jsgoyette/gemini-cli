package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/jsgoyette/gemini"
)

const (
	RETRIES_MAX = 50

	ERROR_API_KEY_MISSING = "Missing API keys. Set GEMINI_API_SANDBOX_KEY " +
		"and GEMINI_API_SANDBOX_SECRET in the environment, or " +
		"GEMINI_API_KEY and GEMINI_API_SECRET for live mode"

	ERROR_AMBIGUOUS_AMOUNT = "Ambiguous use of both amt and base-amt flags"
	ERROR_INVALID_AMOUNT   = "Amount or Child Amount must be above 0"
	ERROR_INVALID_PRICE    = "Price must be above 0"
	ERROR_MAX_RETRIES      = "Max retries"
	ERROR_NO_ASKS          = "No asks in book"
	ERROR_NO_BIDS          = "No bids in book"

	HELP_MSG = `
Usage:

  gemini-cli [command] [flags...]

Commands:
  active
    	List active orders
  balances
    	Get fund balances
  book
    	Get order book
  cancel
    	Cancel active order by txid
  cancel-all
    	Cancel all active orders
  help
    	Show help dialog
  limit
    	Create a limit order
  market
    	Create a market order
  status
    	Get status of active order
  ticker
    	Get ticker
  trades
    	List past trades

Flags:
`
)

var (
	gemini_api_key    string
	gemini_api_secret string

	amount     *float64
	bps        *int
	baseAmount *float64
	date       *string
	lim        *int
	live       *bool
	mkt        *string
	price      *float64
	side       *string
	timestamp  *int64
	txid       *string
	useJson    *bool

	g *gemini.Api
)

var red = color.New(color.FgRed).SprintFunc()
var blue = color.New(color.FgHiBlue).SprintFunc()
var boldWhite = color.New(color.FgWhite).Add(color.Bold).SprintFunc()

var flagset = flag.NewFlagSet("", flag.ExitOnError)

func init() {
	amount = flagset.Float64("amt", 0, "Amount of quote currency")
	bps = flagset.Int("bps", 25, "Fee Basis points")
	baseAmount = flagset.Float64("base-amt", 0, "Amount of base currency")
	date = flagset.String("date", "", "Date (in format of YYYY-MM-DD) for date query")
	lim = flagset.Int("limit", 20, "Limit for list query")
	live = flagset.Bool("live", false, "Live mode: true, false (default false)")
	mkt = flagset.String("mkt", "btcusd", "Market: btcusd, ethusd, ethbtc")
	price = flagset.Float64("price", 0, "Price of parent denomination")
	side = flagset.String("side", "buy", "Side: buy, sell")
	timestamp = flagset.Int64("time", 0, "Timestamp (with milliseconds) for date query")
	txid = flagset.String("txid", "", "Id of order")
	useJson = flagset.Bool("json", false, "Return in JSON format: true, false (default false)")
}

func main() {

	if len(os.Args) < 2 {
		help()
		return
	}

	flagset.Parse(os.Args[2:])

	err := verifyApiKeys(*live)
	if err != nil {
		exitWithError(err)
	}

	g = gemini.New(*live, gemini_api_key, gemini_api_secret)

	if *baseAmount > 0 && *amount > 0 {
		exitWithError(errors.New(ERROR_AMBIGUOUS_AMOUNT))
	}

	switch cmd := os.Args[1]; cmd {
	case "active":
		active(*useJson)
	case "balances":
		balances(*useJson)
	case "book":
		book(*mkt, *lim, *useJson)
	case "cancel":
		cancel(*txid, *useJson)
	case "cancel-all":
		cancelAll(*useJson)
	case "limit":
		limit(*mkt, *side, *amount, *baseAmount, *bps, *price, *useJson)
	case "market":
		market(*mkt, *side, *amount, *baseAmount, *bps, *useJson)
	case "status":
		status(*txid, *useJson)
	case "ticker":
		ticker(*mkt, *useJson)
	case "trades":
		if *date != "" {
			*timestamp, err = getTimeFromDate(*date)
			if err != nil {
				exitWithError(err)
			}
		}
		trades(*mkt, *lim, *timestamp, *useJson)
	default:
		help()
	}

}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
	os.Exit(1)
}

func getTimeFromDate(date string) (int64, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, err
	}

	return t.UnixNano() / int64(time.Millisecond), nil
}

func round(v float64, decimals int) float64 {
	var pow float64 = 1
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int((v*pow)+0.5)) / pow
}

func verifyApiKeys(live bool) error {

	if live {
		gemini_api_key = os.Getenv("GEMINI_API_KEY")
		gemini_api_secret = os.Getenv("GEMINI_API_SECRET")
	} else {
		gemini_api_key = os.Getenv("GEMINI_API_SANDBOX_KEY")
		gemini_api_secret = os.Getenv("GEMINI_API_SANDBOX_SECRET")
	}

	if gemini_api_key == "" || gemini_api_secret == "" {
		return errors.New(ERROR_API_KEY_MISSING)
	}

	return nil
}

func active(useJson bool) {
	activeOrders, err := g.ActiveOrders()
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	if useJson {
		chars, _ := json.Marshal(activeOrders)
		fmt.Println(string(chars))
		return
	}

	for idx, order := range activeOrders {
		printOrder(order)
		if idx < len(activeOrders)-1 {
			fmt.Println("")
		}
	}
}

func balances(useJson bool) {
	balances, err := g.Balances()
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(balances)
		fmt.Println(string(chars))
		return
	}

	for _, fund := range balances {
		fmt.Printf("%s: %v\n", blue(fund.Currency), fund.Amount)
	}
}

func book(mkt string, lim int, useJson bool) {

	book, err := g.OrderBook(mkt, lim, lim)
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(book)
		fmt.Println(string(chars))
		return
	}

	for i := len(book.Asks) - 1; i >= 0; i-- {
		ask := book.Asks[i]

		askAmount := fmt.Sprintf("%.8f", ask.Amount)
		askPrice := fmt.Sprintf("%.8f", ask.Price)

		if i == 0 {
			fmt.Printf("%s\t%s\n", boldWhite(askPrice), askAmount)
		} else {
			fmt.Printf("%s\t%s\n", blue(askPrice), askAmount)
		}
	}

	fmt.Println("")

	for i, l := 0, len(book.Bids); i < l; i++ {
		bid := book.Bids[i]

		bidAmount := fmt.Sprintf("%.8f", bid.Amount)
		bidPrice := fmt.Sprintf("%.8f", bid.Price)

		if i == 0 {
			fmt.Printf("%s\t%s\n", boldWhite(bidPrice), bidAmount)
		} else {
			fmt.Printf("%s\t%s\n", blue(bidPrice), bidAmount)
		}
	}

}

func cancel(txid string, useJson bool) {
	order, err := g.CancelOrder(txid)
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(order)
		fmt.Println(string(chars))
		return
	}

	printOrder(order)
}

func cancelAll(useJson bool) {
	res, err := g.CancelAll()
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(res)
		fmt.Println(string(chars))
		return
	}

	fmt.Printf("%s: %+v\n", blue("Cancelled Orders"), res.Details.CancelledOrders)
	fmt.Printf("%s: %+v\n", blue("Rejected Orders"), res.Details.CancelRejects)
}

func help() {
	fmt.Println(HELP_MSG)
	flagset.PrintDefaults()
}

func limit(mkt, side string, amount, baseAmount float64, bps int, price float64, useJson bool) {

	if amount <= 0.0 && baseAmount <= 0.0 {
		exitWithError(errors.New(ERROR_INVALID_AMOUNT))
	}

	if price <= 0.0 {
		exitWithError(errors.New(ERROR_INVALID_PRICE))
	}

	decimals := 8
	if mkt != "btcusd" {
		decimals = 6
	}

	feeRatio := getFeeRatio(bps)

	if side == "buy" {
		amount -= amount * feeRatio
	} else {
		amount += amount * feeRatio
	}

	var btcAmount float64

	if amount > 0 {
		btcAmount = round(amount/price, decimals)
	} else {
		btcAmount = round(baseAmount, decimals)
	}

	// commit trade
	order, err := g.NewOrder(mkt, "", btcAmount, price, side, []string{"maker-or-cancel"})
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(order)
		fmt.Println(string(chars))
		return
	}

	printOrder(order)
}

func getOrderBookEntry(mkt, side string) gemini.BookEntry {
	// grab order book to get current prices
	book, err := g.OrderBook(mkt, 1, 1)
	if err != nil {
		exitWithError(err)
	}

	if side == "buy" {
		if len(book.Asks) < 1 {
			exitWithError(errors.New(ERROR_NO_ASKS))
		}

		return book.Asks[0]
	}

	if len(book.Bids) < 1 {
		exitWithError(errors.New(ERROR_NO_BIDS))
	}

	return book.Bids[0]
}

func getFeeRatio(bps int) float64 {
	return float64(bps) / 10000
}

func market(mkt, side string, amount, baseAmount float64, bps int, useJson bool) {

	if amount <= 0.0 && baseAmount <= 0.0 {
		exitWithError(errors.New(ERROR_INVALID_AMOUNT))
	}

	retries := 0
	executedAmt := 0.0
	orders := make([]gemini.Order, 0, 10)

	decimals := 8
	if mkt != "btcusd" {
		decimals = 6
	}

	feeRatio := getFeeRatio(bps)

	if side == "buy" {
		amount -= amount * feeRatio
	} else {
		amount += amount * feeRatio
	}

	for {

		if retries == RETRIES_MAX {
			exitWithError(errors.New(ERROR_MAX_RETRIES))
		}

		var fillAmount float64
		if amount > 0 {
			fillAmount = amount
		} else {
			fillAmount = baseAmount
		}

		if executedAmt > 0 {
			fillAmount = fillAmount - executedAmt
		}

		bookEntry := getOrderBookEntry(mkt, side)

		var btcAmount float64
		if amount > 0 {
			btcAmount = round(fillAmount/bookEntry.Price, decimals)
		} else {
			btcAmount = round(fillAmount, decimals)
		}

		if bookEntry.Amount < btcAmount {
			btcAmount = round(bookEntry.Amount, decimals)
		}

		// commit trade
		order, err := g.NewOrder(mkt, "", btcAmount, bookEntry.Price, side, []string{"immediate-or-cancel"})
		if err != nil {
			exitWithError(err)
		}

		if useJson {
			orders = append(orders, order)
		} else {
			printOrder(order)
		}

		// fmt.Printf("%+v\n", order)
		if amount > 0 {
			executedAmt += order.ExecutedAmount * order.AvgExecutionPrice
		} else {
			executedAmt += order.ExecutedAmount
		}

		if (amount > 0 && executedAmt >= amount) || (baseAmount > 0 && executedAmt >= baseAmount) {
			if useJson {
				chars, _ := json.Marshal(orders)
				fmt.Println(string(chars))
			}
			return
		}

		fmt.Println("")
		retries++
	}
}

func printOrder(order gemini.Order) {
	// log.Printf("Trade Created: %+v", order)
	fmt.Printf("%s:\t\t%s\n", blue("OrderId"), boldWhite(order.OrderId))
	fmt.Printf("%s:\t\t\t%s\n", blue("Symbol"), order.Symbol)
	fmt.Printf("%s:\t\t\t%s\n", blue("Side"), order.Side)
	fmt.Printf("%s:\t\t\t%.8f\n", blue("Price"), order.Price)
	fmt.Printf("%s:\t\t%.8f\n", blue("OriginalAmount"), order.OriginalAmount)
	fmt.Printf("%s:\t\t%.8f\n", blue("ExecutedAmount"), order.ExecutedAmount)
	fmt.Printf("%s:\t%.8f\n", blue("RemainingAmount"), order.RemainingAmount)
	fmt.Printf("%s:\t%.8f\n", blue("AvgExecutionPrice"), order.AvgExecutionPrice)
	fmt.Printf("%s:\t\t\t%v\n", blue("IsLive"), order.IsLive)
	fmt.Printf("%s:\t\t%v\n", blue("IsCancelled"), order.IsCancelled)
}

func printTrade(trade gemini.Trade) {
	fmt.Printf("%s:\t%s\n", blue("OrderId"), boldWhite(trade.OrderId))
	fmt.Printf("%s:\t%v\n", blue("Timestamp"), trade.Timestamp)
	fmt.Printf("%s:\t\t%s\n", blue("Type"), trade.Type)
	fmt.Printf("%s:\t\t%.8f\n", blue("Price"), trade.Price)
	fmt.Printf("%s:\t\t%.8f\n", blue("Amount"), trade.Amount)
	fmt.Printf("%s:\t%.8f\n", blue("FeeAmount"), trade.FeeAmount)
	fmt.Printf("%s:\t\t%v\n", blue("Maker"), !trade.Aggressor)
}

func status(txid string, useJson bool) {
	order, err := g.OrderStatus(txid)
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(order)
		fmt.Println(string(chars))
		return
	}

	printOrder(order)
}

func ticker(mkt string, useJson bool) {
	t, err := g.Ticker(mkt)
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(t)
		fmt.Println(string(chars))
		return
	}

	fmt.Printf("%s:\t%s\n", blue("Bid"), boldWhite(t.Bid))
	fmt.Printf("%s:\t%s\n", blue("Ask"), boldWhite(t.Ask))
	fmt.Printf("%s:\t%.8f\n", blue("Last"), t.Last)
	fmt.Printf("%s:\t%v\n", blue("Volume"), t.Volume.BTC)
}

func trades(mkt string, lim int, timestamp int64, useJson bool) {
	pastTrades, err := g.PastTrades(mkt, lim, timestamp)
	if err != nil {
		exitWithError(err)
	}

	if useJson {
		chars, _ := json.Marshal(pastTrades)
		fmt.Println(string(chars))
		return
	}

	for idx, trade := range pastTrades {
		printTrade(trade)
		if idx < len(pastTrades)-1 {
			fmt.Println("")
		}
	}

}
