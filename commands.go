package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jsgoyette/gemini"
	"github.com/urfave/cli"
)

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

		if c.Bool("unsafe") == false {
			printOrder(order)
			return nil
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
