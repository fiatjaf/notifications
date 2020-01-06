package main

import (
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/kballard/go-shellquote"
)

const USAGE = `
Usage:
  /help
  /start [<jqfilter>]
  /setfilter <channel> <jqfilter>
  /subscribe <channel>
  /list
  /delete (all | <channel>)
`

var parser = &docopt.Parser{HelpHandler: func(err error, usage string) {}}

func parseCommand(text string) (o docopt.Opts, err error) {
	text = strings.Replace(text, "/", "", 1)

	argv, err := shellquote.Split(text)
	if err != nil {
		return
	}

	return parser.ParseArgs(
		strings.Replace(USAGE, "/", "incomingnotificationsbot ", -1),
		argv, "")
}
