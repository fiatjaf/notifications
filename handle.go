package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/itchyny/gojq"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

func handle(upd tgbotapi.Update) {
	if upd.Message != nil {
		handleMessage(upd.Message)
	}
}

func handleMessage(message *tgbotapi.Message) {
	log := log.With().Int("user", message.From.ID).Str("name", message.From.UserName).
		Int64("chat", message.Chat.ID).
		Str("command", message.Text).
		Logger()

	if message.Text[0] != '/' {
		return
	}

	opts, err := parseCommand(message.Text)
	if err != nil {
		log.Debug().Err(err).Msg("invalid")

		sendMessage(message.Chat.ID,
			strings.Replace(
				strings.Replace(USAGE,
					">", "&gt;", -1),
				"<", "&lt;", -1),
		)
		return
	}
	log.Debug().Str("command", message.Text).Msg("command")

	defer func() {
		if err != nil {
			sendMessage(message.Chat.ID, "Error: "+err.Error())
		}
	}()

	switch {
	case opts["help"].(bool), opts["start"].(bool):
		sendMessage(message.Chat.ID, `
Call <code>/url &lt;jqfilter&gt;</code> to get your URL to receive incoming notifications.

To be notified of anything, send a webhook to that URL.

Any data you send to that URL (either JSON, querystring, text/plain or form data) will be turned into JSON and passed to a <a href="https://stedolan.github.io/jq/manual/">jq filter</a>.

The default filter is <code>.</code>, which just gives you just the full body data from the webhook.
        `)
	case opts["url"].(bool):
		filter, err := opts.String("<jqfilter>")
		if err != nil {
			filter = "."
		} else {
			_, err = gojq.Parse(filter)
			if err != nil {
				err = fmt.Errorf("error parsing filter: %w", err)
				sendMessage(message.Chat.ID, err.Error())
				return
			}
		}

		id, err := h.EncodeInt64([]int64{message.Chat.ID})
		if err != nil {
			log.Error().Err(err).Int64("chatid", message.Chat.ID).
				Msg("failed to encode chat id hashid")
			sendMessage(message.Chat.ID, "unexpected error encoding chat id")
			return
		}

		query := ""
		if filter != "." {
			query = "?" + (url.Values{"jq": {filter}}).Encode()
		}
		text := fmt.Sprintf(`Send HTTP requests to

<code>%s/w/%s%s</code>`,
			s.ServiceURL, id, query)
		sendMessage(message.Chat.ID, text)
	}
}
