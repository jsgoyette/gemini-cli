package main

import (
	"errors"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/jsgoyette/gemini"
	"github.com/urfave/cli"
)

const (
	ERROR_API_KEY_MISSING = "Missing API keys. Set GEMINI_API_SANDBOX_KEY " +
		"and GEMINI_API_SANDBOX_SECRET in the environment, or " +
		"GEMINI_API_KEY and GEMINI_API_SECRET for live mode"

	ERROR_AMBIGUOUS_AMOUNT = "Ambiguous use of both amt and base-amt flags"
	ERROR_INVALID_AMOUNT   = "Amount or Base Amount must be above 0"
	ERROR_INVALID_PRICE    = "Price must be above 0"
	ERROR_MAX_RETRIES      = "Max retries"
	ERROR_NO_ASKS          = "No asks in book"
	ERROR_NO_BIDS          = "No bids in book"

	RETRIES_MAX = 50
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
