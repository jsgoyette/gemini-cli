package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jsgoyette/gemini"
)

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

func printError(err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
	fmt.Fprintf(os.Stderr, "")
	return
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
