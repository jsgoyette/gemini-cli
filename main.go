package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/jsgoyette/gemini"
	"github.com/urfave/cli"
)

const (
	RETRIES_MAX = 50

	ERROR_API_KEY_MISSING = "Missing API keys. Set GEMINI_API_SANDBOX_KEY " +
		"and GEMINI_API_SANDBOX_SECRET in the environment, or " +
		"GEMINI_API_KEY and GEMINI_API_SECRET for live mode"

	ERROR_AMBIGUOUS_AMOUNT = "Ambiguous use of both amt and base-amt flags"
	ERROR_INVALID_AMOUNT   = "Amount or Base Amount must be above 0"
	ERROR_INVALID_PRICE    = "Price must be above 0"
	ERROR_MAX_RETRIES      = "Max retries"
	ERROR_NO_ASKS          = "No asks in book"
	ERROR_NO_BIDS          = "No bids in book"
)

var (
	gemini_api_key    string
	gemini_api_secret string

	g *gemini.Api

	amtFlag = cli.Float64Flag{
		Name:  "amt",
		Value: 0,
		Usage: "Amount of quote currency",
	}
	bpsFlag = cli.IntFlag{
		Name:  "bps",
		Value: 25,
		Usage: "Fee Basis points",
	}
	baseAmtFlag = cli.Float64Flag{
		Name:  "base-amt",
		Value: 0,
		Usage: "Amount of base currency",
	}
	dateFlag = cli.StringFlag{
		Name:  "date",
		Value: "",
		Usage: "Date (in format of YYYY-MM-DD) for date query",
	}
	jsonFlag = cli.BoolFlag{
		Name:  "json",
		Usage: "Return in JSON format: true, false (default false)",
	}
	limitFlag = cli.IntFlag{
		Name:  "lim",
		Value: 20,
		Usage: "Limit for list query",
	}
	liveFlag = cli.BoolFlag{
		Name:  "live",
		Usage: "Live mode: true, false (default false)",
	}
	mktFlag = cli.StringFlag{
		Name:  "mkt",
		Value: "btcusd",
		Usage: "Market: btcusd, ethusd, ethbtc",
	}
	priceFlag = cli.Float64Flag{
		Name:  "price",
		Value: 0,
		Usage: "Price of parent denomination",
	}
	sideFlag = cli.StringFlag{
		Name:  "side",
		Value: "buy",
		Usage: "Side: buy, sell",
	}
	timeFlag = cli.Int64Flag{
		Name:  "time",
		Value: 0,
		Usage: "Timestamp (with milliseconds) for date query",
	}
	txidFlag = cli.StringFlag{
		Name:  "txid",
		Value: "",
		Usage: "Id of order",
	}

	commands = []cli.Command{
		{
			Name:    "active",
			Aliases: []string{"a"},
			Usage:   "List active orders",
			Action:  active,
			Flags:   []cli.Flag{jsonFlag},
		},
		{
			Name:    "balances",
			Aliases: []string{"b"},
			Usage:   "Get fund balances",
			Action:  balances,
			Flags:   []cli.Flag{jsonFlag},
		},
		{
			Name:    "book",
			Aliases: []string{"bk"},
			Usage:   "Get order book",
			Action:  book,
			Flags:   []cli.Flag{mktFlag, limitFlag, jsonFlag},
		},
		{
			Name:    "cancel",
			Aliases: []string{"c"},
			Usage:   "Cancel active order by txid",
			Action:  cancel,
			Flags:   []cli.Flag{txidFlag, jsonFlag},
		},
		{
			Name:    "cancel-all",
			Aliases: []string{"ca"},
			Usage:   "Cancel all active orders",
			Action:  cancelAll,
			Flags:   []cli.Flag{jsonFlag},
		},
		{
			Name:    "limit",
			Aliases: []string{"l"},
			Usage:   "Create a limit order",
			Action:  limit,
			Flags: []cli.Flag{
				amtFlag,
				baseAmtFlag,
				bpsFlag,
				jsonFlag,
				mktFlag,
				priceFlag,
				sideFlag,
			},
			Before: beforeTransaction,
		},
		{
			Name:    "market",
			Aliases: []string{"m"},
			Usage:   "Create a market order",
			Action:  market,
			Flags: []cli.Flag{
				amtFlag,
				baseAmtFlag,
				bpsFlag,
				jsonFlag,
				mktFlag,
				sideFlag,
			},
			Before: beforeTransaction,
		},
		{
			Name:    "status",
			Aliases: []string{"s"},
			Usage:   "Get status of active order",
			Action:  status,
			Flags:   []cli.Flag{txidFlag, jsonFlag},
		},
		{
			Name:    "ticker",
			Aliases: []string{"tr"},
			Usage:   "Get ticker",
			Action:  ticker,
			Flags:   []cli.Flag{mktFlag, jsonFlag},
		},
		{
			Name:    "trades",
			Aliases: []string{"t"},
			Usage:   "List past trades",
			Action:  trades,
			Flags: []cli.Flag{
				dateFlag,
				jsonFlag,
				limitFlag,
				mktFlag,
				timeFlag,
			},
			Before: beforeTrade,
		},
	}

	red       = color.New(color.FgRed).SprintFunc()
	blue      = color.New(color.FgHiBlue).SprintFunc()
	boldWhite = color.New(color.FgWhite).Add(color.Bold).SprintFunc()
)

func main() {
	app := cli.NewApp()

	app.Usage = "Gemini API utility"
	app.UsageText = "gemini-cli [global options] command [command options]"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{liveFlag}
	app.Before = beforeApp
	app.Commands = commands

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Run(os.Args)
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
	fmt.Fprintf(os.Stderr, "")
	return
}

func beforeApp(c *cli.Context) error {
	live := c.Bool("live")

	err := verifyApiKeys(live)
	if err != nil {
		printError(err)
		return err
	}

	g = gemini.New(live, gemini_api_key, gemini_api_secret)

	return nil
}

func beforeTrade(c *cli.Context) error {
	date := c.String("date")

	if date != "" {
		t, err := getTimeFromDate(date)
		if err != nil {
			printError(err)
			return err
		}

		c.Set("time", string(t))
	}

	return nil
}

func beforeTransaction(c *cli.Context) error {
	if c.Float64("base-amt") > 0 && c.Float64("amt") > 0 {
		err := errors.New(ERROR_AMBIGUOUS_AMOUNT)
		printError(err)
		return err
	}
	return nil
}

func getFeeRatio(bps int) float64 {
	return float64(bps) / 10000
}

func getOrderBookEntry(mkt, side string) (*gemini.BookEntry, error) {
	book, err := g.OrderBook(mkt, 1, 1)

	if err != nil {
		return nil, err
	}

	if side == "buy" {
		if len(book.Asks) < 1 {
			return nil, errors.New(ERROR_NO_ASKS)
		}

		return &book.Asks[0], nil
	}

	if len(book.Bids) < 1 {
		return nil, errors.New(ERROR_NO_BIDS)
	}

	return &book.Bids[0], nil
}

func getTimeFromDate(date string) (int64, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, err
	}

	return t.UnixNano() / int64(time.Millisecond), nil
}

func printOrder(order gemini.Order) {
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

func active(c *cli.Context) error {
	activeOrders, err := g.ActiveOrders()
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(activeOrders)
		fmt.Println(string(chars))
		return nil
	}

	for idx, order := range activeOrders {
		printOrder(order)
		if idx < len(activeOrders)-1 {
			fmt.Println("")
		}
	}

	return nil
}

func balances(c *cli.Context) error {
	balances, err := g.Balances()
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(balances)
		fmt.Println(string(chars))
		return nil
	}

	for _, fund := range balances {
		fmt.Printf("%s: %v\n", blue(fund.Currency), fund.Amount)
	}

	return nil
}

func book(c *cli.Context) error {

	mkt := c.String("mkt")
	lim := c.Int("lim")

	book, err := g.OrderBook(mkt, lim, lim)
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(book)
		fmt.Println(string(chars))
		return nil
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

	return nil
}

func cancel(c *cli.Context) error {
	order, err := g.CancelOrder(c.String("txid"))
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(order)
		fmt.Println(string(chars))
		return nil
	}

	printOrder(order)
	return nil
}

func cancelAll(c *cli.Context) error {
	res, err := g.CancelAll()
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(res)
		fmt.Println(string(chars))
		return nil
	}

	fmt.Printf("%s: %+v\n", blue("Cancelled Orders"), res.Details.CancelledOrders)
	fmt.Printf("%s: %+v\n", blue("Rejected Orders"), res.Details.CancelRejects)

	return nil
}

func limit(c *cli.Context) error {

	amount := c.Float64("amt")
	baseAmount := c.Float64("base-amt")
	bps := c.Int("bps")
	mkt := c.String("mkt")
	price := c.Float64("price")
	side := c.String("side")

	if amount <= 0.0 && baseAmount <= 0.0 {
		err := errors.New(ERROR_INVALID_AMOUNT)
		printError(err)
		return err
	}

	if price <= 0.0 {
		err := errors.New(ERROR_INVALID_PRICE)
		printError(err)
		return err
	}

	var btcAmount float64

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

	if amount > 0 {
		btcAmount = round(amount/price, decimals)
	} else {
		btcAmount = round(baseAmount, decimals)
	}

	// commit trade
	order, err := g.NewOrder(mkt, "", btcAmount, price, side, []string{"maker-or-cancel"})
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(order)
		fmt.Println(string(chars))
		return nil
	}

	printOrder(order)
	return nil
}

func market(c *cli.Context) error {

	amount := c.Float64("amt")
	baseAmount := c.Float64("base-amt")
	bps := c.Int("bps")
	mkt := c.String("mkt")
	side := c.String("side")

	if amount <= 0.0 && baseAmount <= 0.0 {
		err := errors.New(ERROR_INVALID_AMOUNT)
		printError(err)
		return err
	}

	retries := 0
	executedAmt := 0.0
	orders := make([]gemini.Order, 0, 10)

	decimals := 8
	if mkt != "btcusd" {
		decimals = 6
	}

	minAmt := 0.000001
	if (mkt == "btcusd" || mkt == "ethusd") && amount > 0 {
		minAmt = 0.01
	}
	if (mkt == "ethbtc" || mkt == "ethusd") && baseAmount > 0 {
		minAmt = 0.0001
	}

	feeRatio := getFeeRatio(bps)

	if side == "buy" {
		amount -= amount * feeRatio
	} else {
		amount += amount * feeRatio
	}

	for {

		if retries == RETRIES_MAX {
			err := errors.New(ERROR_MAX_RETRIES)
			printError(err)
			return err
		}

		var fillAmount, btcAmount float64

		bookEntry, err := getOrderBookEntry(mkt, side)
		if err != nil {
			printError(err)
			return err
		}

		if amount > 0 {
			fillAmount = amount
		} else {
			fillAmount = baseAmount
		}

		if executedAmt > 0 {
			fillAmount = fillAmount - executedAmt
		}

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
			printError(err)
			return err
		}

		if c.Bool("json") {
			orders = append(orders, order)
		} else {
			printOrder(order)
		}

		if amount > 0 {
			executedAmt += order.ExecutedAmount * order.AvgExecutionPrice
		} else {
			executedAmt += order.ExecutedAmount
		}

		if (amount > 0 && executedAmt >= amount-minAmt) || (baseAmount > 0 && executedAmt >= baseAmount-minAmt) {
			if c.Bool("json") {
				chars, _ := json.Marshal(orders)
				fmt.Println(string(chars))
			}
			return nil
		}

		fmt.Println("")
		retries++
	}
}

func status(c *cli.Context) error {
	order, err := g.OrderStatus(c.String("txid"))
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(order)
		fmt.Println(string(chars))
		return nil
	}

	printOrder(order)

	return nil
}

func ticker(c *cli.Context) error {
	t, err := g.Ticker(c.String("mkt"))
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(t)
		fmt.Println(string(chars))
		return nil
	}

	fmt.Printf("%s:\t%s\n", blue("Bid"), boldWhite(t.Bid))
	fmt.Printf("%s:\t%s\n", blue("Ask"), boldWhite(t.Ask))
	fmt.Printf("%s:\t%.8f\n", blue("Last"), t.Last)
	fmt.Printf("%s:\t%v\n", blue("Volume"), t.Volume.BTC)

	return nil
}

func trades(c *cli.Context) error {
	mkt := c.String("mkt")
	lim := c.Int("lim")
	timestamp := c.Int64("time")

	pastTrades, err := g.PastTrades(mkt, lim, timestamp)
	if err != nil {
		printError(err)
		return err
	}

	if c.Bool("json") {
		chars, _ := json.Marshal(pastTrades)
		fmt.Println(string(chars))
		return nil
	}

	for idx, trade := range pastTrades {
		printTrade(trade)
		if idx < len(pastTrades)-1 {
			fmt.Println("")
		}
	}

	return nil
}
