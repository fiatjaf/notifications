package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/speps/go-hashids"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type Settings struct {
	BotToken   string `envconfig:"BOT_TOKEN" required:"true"`
	Host       string `envconfig:"HOST" required:"true"`
	Port       string `envconfig:"PORT" required:"true"`
	ServiceURL string `envconfig:"SERVICE_URL" required:"true"`
}

var s Settings
var h *hashids.HashID
var bot *tgbotapi.BotAPI
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var router = mux.NewRouter()

func main() {
	err := envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	// hashids
	hd := hashids.NewData()
	hd.Salt = "this is my salt"
	h, err = hashids.NewWithData(hd)
	if err != nil {
		log.Fatal().Err(err).Msg("hashids initialization")
	}

	// bot stuff
	bot, err = tgbotapi.NewBotAPI(s.BotToken)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	log.Info().Str("username", bot.Self.UserName).Msg("telegram bot authorized")

	// http server
	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://t.me/incomingnotificationsbot", 302)
	})
	router.Path("/" + bot.Token).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadAll(r.Body)
		var update tgbotapi.Update
		json.Unmarshal(bytes, &update)
		handle(update)
	})
	router.PathPrefix("/w/{id}").HandlerFunc(handleWebhook)

	go func() {
		time.Sleep(1 * time.Second)
		// set webhook
		_, err = bot.SetWebhook(tgbotapi.NewWebhook(s.ServiceURL + "/" + bot.Token))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to set webhook")
		}
		_, err := bot.GetWebhookInfo()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get webhook info")
		}
	}()

	// listen http
	log.Info().Str("port", s.Port).Msg("listening")
	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + s.Port,
		WriteTimeout: 300 * time.Second,
		ReadTimeout:  300 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error().Err(err).Msg("error serving http")
	}
}
