package main

import (
	"github.com/docopt/docopt-go"
	"github.com/kballard/go-shellquote"
)

const USAGE = `incomingnotificationsbot

Usage:
  incomingnotificationsbot help
  incomingnotificationsbot start [<jq_filter>]
  incomingnotificationsbot subscribe <channel>
  incomingnotificationsbot list
  incomingnotificationsbot delete [all | <channel>]
`

func parseCommand(text string) (o docopt.Opts, err error) {
	argv, err := shellquote.Split(text)
	if err != nil {
		return
	}

	return docopt.ParseArgs(USAGE, argv, "")
}
