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

	red       = color.New(color.FgRed).SprintFunc()
	blue      = color.New(color.FgHiBlue).SprintFunc()
	boldWhite = color.New(color.FgWhite).Add(color.Bold).SprintFunc()
)

func main() {
	app := cli.NewApp()

	app.Usage = "CLI for the Gemini Bitcoin exchange API"
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
