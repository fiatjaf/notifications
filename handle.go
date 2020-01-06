package main

import (
	"fmt"
	"strings"

	"github.com/hoisie/mustache"
	"github.com/itchyny/gojq"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

func handle(upd tgbotapi.Update) {
	if upd.Message != nil {
		handleMessage(upd.Message)
	}
}

func handleMessage(message *tgbotapi.Message) {
	if message.Text[0] != '/' {
		return
	}

	opts, err := parseCommand(message.Text)
	if err != nil {
		log.Debug().Str("command", message.Text).Err(err).Msg("invalid")
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
	case opts["help"].(bool):
		sendMessage(message.Chat.ID, `
Call <code>/start &lt;jqfilter&gt;</code> to generate a channel to receive incoming notifications.

To be notified of anything, send a webhook to the URL that will appear.

Any data you send to that URL (either JSON, querystring, text/plain or form data) will be turned into JSON and passed to the <a href="https://stedolan.github.io/jq/manual/">jq filter</a> as

<pre>
{
  "channel": "channel_id",
  "headers": {...},
  "data": {...}
}
</pre>

The default filter is <code>.data</code>, which gives you just the raw data from the webhook.
        `)
	case opts["start"].(bool):
		filter, err := opts.String("<jqfilter>")
		if err != nil {
			filter = ".data"
		} else {
			_, err = gojq.Parse(filter)
			if err != nil {
				err = fmt.Errorf("error parsing filter: %w", err)
				return
			}
		}

		tx, err := pg.Beginx()
		if err != nil {
			return
		}
		defer tx.Rollback()

		var id string
		err = tx.Get(&id, `INSERT INTO channel (jq) VALUES ($1) RETURNING id`, filter)
		if err != nil {
			return
		}
		_, err = tx.Exec(`INSERT INTO subscription (channel, chat_id) VALUES ($1, $2)`,
			id, message.Chat.ID)
		if err != nil {
			return
		}

		err = tx.Commit()
		if err != nil {
			return
		}

		text := mustache.Render(`Channel <b>{{Id}}</b> created. 
          Send HTTP requests to <code>{{ServiceURL}}/n/{{Id}}</code>.
        `, map[string]interface{}{
			"ServiceURL": s.ServiceURL,
			"Id":         id,
			"Filter":     filter,
		})
		sendMessage(message.Chat.ID, text)
	case opts["delete"].(bool):
		if channel, err := opts.String("<channel>"); err == nil {
			_, err = pg.Exec(`
              DELETE FROM subscription
              WHERE chat_id = $1 AND channel = $2
            `, message.Chat.ID, channel)
		} else {
			_, err = pg.Exec(`
              DELETE FROM subscription
              WHERE chat_id = $1
            `, message.Chat.ID)
		}
		if err != nil {
			return
		}

		sendMessage(message.Chat.ID, "Deleted.")

		go pg.Exec(`
          DELETE FROM channel WHERE id IN (
            SELECT channel.id FROM channel
            LEFT OUTER JOIN subscription ON channel.id = subscription.channel
            WHERE subscription.chat_id IS NULL
          )
        `)
	case opts["subscribe"].(bool):
		channel, _ := opts.String("<channel>")
		_, err = pg.Exec(`
          INSERT INTO subscription (channel, chat_id)
          VALUES ($1, $2)
          ON CONFLICT (channel, chat_id) DO NOTHING
        `,
			channel, message.Chat.ID)
		if err != nil {
			return
		}

		sendMessage(message.Chat.ID, "Subscribed to channel <code>"+channel+"</code>.")
	case opts["list"].(bool):
		var channels []struct {
			Id string `db:"id"`
			JQ string `db:"jq"`
		}
		err = pg.Select(&channels, `
          SELECT id, jq FROM channel
          INNER JOIN subscription ON subscription.channel = channel.id
          WHERE chat_id = $1
          ORDER BY channel.id
        `, message.Chat.ID)

		text := mustache.Render(`<b>Subscribed to channels:</b>
{{#Channels}}
- <u>{{Id}}</u>: <code>{{ServiceURL}}/n/{{Id}}</code> (<code>{{JQ}}</code>){{/Channels}}{{^Channels}}No subscriptions.{{/Channels}}
        `, map[string]interface{}{
			"ServiceURL": s.ServiceURL,
			"Channels":   channels,
		})
		sendMessage(message.Chat.ID, text)
	}
}
