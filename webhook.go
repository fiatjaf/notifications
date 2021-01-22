package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/itchyny/gojq"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	var (
		id      string = mux.Vars(r)["id"]
		filter  string
		chatIds []int64
		data    interface{}
		headers map[string]string
	)

	err := pg.Get(&filter, `SELECT jq FROM channel WHERE id = $1`, id)
	if err != nil {
		log.Warn().Err(err).Str("channel", id).Msg("failed to fetch filter")
		http.Error(w, "", 500)
		return
	}

	f, _ := gojq.Parse(filter)

	err = pg.Select(&chatIds, `
      SELECT chat_id
      FROM subscription
      WHERE channel = $1
    `, id)
	if err != nil {
		log.Warn().Err(err).Str("channel", id).Msg("failed to fetch chat ids")
		http.Error(w, "", 500)
		return
	}

	log.Debug().Str("filter", filter).Str("channel", id).Interface("chats", chatIds).
		Msg("dispatching notification")

	err = r.ParseForm()
	if err != nil {
		log.Warn().Err(err).Msg("parseform")
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)

	datamap := make(map[string]interface{})
	err = json.Unmarshal(body, &datamap)

	if err != nil {
		contentType := r.Header.Get("Content-Type")
		if contentType == "application/x-www-form-urlencoded" ||
			contentType == "application/json" ||
			contentType == "multipart/form-data" ||
			(len(body) == 0 && len(r.URL.RawQuery) > 0) {
			datamap = make(map[string]interface{})
		} else {
			data = string(body)
			goto gotdata
		}
	}

	for k, v := range r.Form {
		if _, exists := datamap[k]; !exists {
			datamap[k] = v[0]
		}
	}
	data = datamap

gotdata:
	headers = make(map[string]string)
	for k, v := range r.Header {
		headers[k] = v[0]
	}

	input := map[string]interface{}{
		"id":      id,
		"headers": headers,
		"path":    r.URL.RawPath,
		"query":   r.URL.RawQuery,
		"data":    data,
	}

	iter := f.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			log.Warn().Err(err).Interface("input", input).Msg("jq error")
			http.Error(w, "", 400)

			for _, chatId := range chatIds {
				sendMessage(chatId, err.Error())
			}

			return
		}

		var text string
		if str, ok := v.(string); ok {
			text = str
		} else {
			res, _ := json.MarshalIndent(v, "", "  ")
			text = string(res)
		}
		for _, chatId := range chatIds {
			sendMessage(chatId, text)
		}
	}
}
