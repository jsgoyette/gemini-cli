package main

import (
	"github.com/urfave/cli"
)

var (
	amtFlag = cli.Float64Flag{
		Name:  "amt, a",
		Value: 0,
		Usage: "Amount of quote currency",
	}
	bpsFlag = cli.IntFlag{
		Name:  "bps",
		Value: 25,
		Usage: "Fee Basis points",
	}
	baseAmtFlag = cli.Float64Flag{
		Name:  "base-amt, A",
		Value: 0,
		Usage: "Amount of base currency",
	}
	dateFlag = cli.StringFlag{
		Name:  "date, T",
		Value: "",
		Usage: "Date (in format of YYYY-MM-DD) for date query",
	}
	jsonFlag = cli.BoolFlag{
		Name:  "json, j",
		Usage: "Return in JSON format: true, false (default false)",
	}
	limitFlag = cli.IntFlag{
		Name:  "lim, l",
		Value: 20,
		Usage: "Limit for list query",
	}
	liveFlag = cli.BoolFlag{
		Name:  "live",
		Usage: "Live mode: true, false (default false)",
	}
	mktFlag = cli.StringFlag{
		Name:  "mkt, m",
		Value: "btcusd",
		Usage: "Market: btcusd, ethusd, ethbtc",
	}
	priceFlag = cli.Float64Flag{
		Name:  "price, p",
		Value: 0,
		Usage: "Price of parent denomination",
	}
	sideFlag = cli.StringFlag{
		Name:  "side, s",
		Value: "buy",
		Usage: "Side: buy, sell",
	}
	timeFlag = cli.Int64Flag{
		Name:  "time, t",
		Value: 0,
		Usage: "Timestamp (with milliseconds) for date query",
	}
	txidFlag = cli.StringFlag{
		Name:  "txid, x",
		Value: "",
		Usage: "Id of order",
	}

	commands = []cli.Command{
		{
			Name:      "active",
			Aliases:   []string{"a"},
			Usage:     "List active orders",
			UsageText: "gemini-cli active [command options]",
			Action:    active,
			Flags:     []cli.Flag{jsonFlag},
		},
		{
			Name:      "balances",
			Aliases:   []string{"b"},
			Usage:     "Get fund balances",
			UsageText: "gemini-cli balances [command options]",
			Action:    balances,
			Flags:     []cli.Flag{jsonFlag},
		},
		{
			Name:      "book",
			Aliases:   []string{"bk"},
			Usage:     "Get order book",
			UsageText: "gemini-cli book [command options]",
			Action:    book,
			Flags:     []cli.Flag{mktFlag, limitFlag, jsonFlag},
		},
		{
			Name:      "cancel",
			Aliases:   []string{"c"},
			Usage:     "Cancel active order by txid",
			UsageText: "gemini-cli cancel [command options]",
			Action:    cancel,
			Flags:     []cli.Flag{txidFlag, jsonFlag},
		},
		{
			Name:      "cancel-all",
			Aliases:   []string{"ca"},
			Usage:     "Cancel all active orders",
			UsageText: "gemini-cli cancel-all [command options]",
			Action:    cancelAll,
			Flags:     []cli.Flag{jsonFlag},
		},
		{
			Name:      "limit",
			Aliases:   []string{"l"},
			Usage:     "Create a limit order",
			UsageText: "gemini-cli limit [command options]",
			Action:    limit,
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
			Name:      "market",
			Aliases:   []string{"m"},
			Usage:     "Create a market order",
			UsageText: "gemini-cli market [command options]",
			Action:    market,
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
			Name:      "status",
			Aliases:   []string{"s"},
			Usage:     "Get status of active order",
			UsageText: "gemini-cli status [command options]",
			Action:    status,
			Flags:     []cli.Flag{txidFlag, jsonFlag},
		},
		{
			Name:      "ticker",
			Aliases:   []string{"tr"},
			Usage:     "Get ticker",
			UsageText: "gemini-cli ticker [command options]",
			Action:    ticker,
			Flags:     []cli.Flag{mktFlag, jsonFlag},
		},
		{
			Name:      "trades",
			Aliases:   []string{"t"},
			Usage:     "List past trades",
			UsageText: "gemini-cli trades [command options]",
			Action:    trades,
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
)
