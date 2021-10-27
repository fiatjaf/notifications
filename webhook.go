package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/itchyny/gojq"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("jq")
	if filter == "" {
		filter = "."
	}

	chatIds, err := h.DecodeInt64WithError(mux.Vars(r)["id"])
	if err != nil || len(chatIds) != 1 {
		log.Warn().Err(err).Str("url", r.URL.String()).
			Msg("failed to decode hashid")
		w.WriteHeader(400)
		return
	}
	chatId := chatIds[0]

	var (
		data    interface{}
		headers map[string]string
	)

	log.Debug().Str("filter", filter).Int64("chat", chatId).
		Msg("dispatching notification")

	if err := r.ParseForm(); err != nil {
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
		"headers": headers,
		"path":    r.URL.RawPath,
		"query":   r.URL.RawQuery,
		"data":    data,
	}

	f, _ := gojq.Parse(".data | " + filter) // TODO set headers etc. as $variables
	iter := f.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			log.Warn().Err(err).Interface("input", input).Msg("jq error")
			http.Error(w, "", 400)
			sendMessage(chatId, err.Error())
			return
		}

		var text string
		if str, ok := v.(string); ok {
			text = str
		} else {
			res, _ := json.MarshalIndent(v, "", "  ")
			text = string(res)
		}
		sendMessage(chatId, text)
	}
}
