package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type Settings struct {
	BotToken    string `envconfig:"BOT_TOKEN" required:"true"`
	PostgresURL string `envconfig:"DATABASE_URL" required:"true"`
	Host        string `envconfig:"HOST" required:"true"`
	Port        string `envconfig:"PORT" required:"true"`
	ServiceURL  string `envconfig:"SERVICE_URL" required:"true"`
}

var err error
var s Settings
var pg *sqlx.DB
var bot *tgbotapi.BotAPI
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var router = mux.NewRouter()

func main() {
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	pg, err = sqlx.Connect("postgres", s.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to postgres")
	}

	// http server
	router.Path("/n/{id}").HandlerFunc(handleWebhook)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://t.me/incomingnotificationsbot", 302)
	})

	log.Info().Str("port", s.Port).Msg("listening")
	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + s.Port,
		WriteTimeout: 300 * time.Second,
		ReadTimeout:  300 * time.Second,
	}
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Error().Err(err).Msg("error serving http")
		}
	}()

	// bot stuff
	bot, err = tgbotapi.NewBotAPI(s.BotToken)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	log.Info().Str("username", bot.Self.UserName).Msg("telegram bot authorized")

	lastTelegramUpdate, err := getLastTelegramUpdate()
	if err != nil {
		log.Fatal().Err(err).Int64("got", lastTelegramUpdate).
			Msg("failed to get lasttelegramupdate")
		return
	}

	u := tgbotapi.NewUpdate(int(lastTelegramUpdate + 1))
	u.Timeout = 600
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Error().Err(err).Msg("telegram getupdates fail")
		return
	}

	for update := range updates {
		lastTelegramUpdate = int64(update.UpdateID)
		go setLastTelegramUpdate(lastTelegramUpdate)
		handle(update)
	}
}
