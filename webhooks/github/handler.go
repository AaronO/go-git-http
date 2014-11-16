package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type WebhookHandler func(eventname string, payload *GitHubPayload) bool

func Handler(secret string, fn WebhookHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		event := req.Header.Get("x-github-event")
		delivery := req.Header.Get("x-github-delivery")
		signature := req.Header.Get("x-hub-signature")

		// Utility funcs
		_fail := func(err error) {
			fail(w, event, err)
		}
		_succeed := func() {
			succeed(w, event)
		}

		// Ensure headers are all there
		if event == "" || delivery == "" || signature == "" {
			_fail(fmt.Errorf("Missing x-github-* and x-hub-* headers"))
			return
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			_fail(err)
			return
		}

		// Check that payload came from github
		// skip check if empty secret provided
		if secret != "" && !IsValidPayload(secret, signature, body) {
			_fail(fmt.Errorf("Payload did not come from GitHub"))
			return
		}

		// Get payload
		payload := GitHubPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			_fail(fmt.Errorf("Could not deserialize payload"))
			return
		}

		// Do something with payload
		success := fn(event, &payload)
		if success {
			_succeed()
		} else {
			_fail(nil)
		}
	})
}

func succeed(w http.ResponseWriter, event string) {
	render(w, PayloadPong{
		Ok:    true,
		Event: event,
	})
}

func fail(w http.ResponseWriter, event string, err error) {
	w.WriteHeader(500)
	render(w, PayloadPong{
		Ok:    false,
		Event: event,
		Error: err,
	})
}

func render(w http.ResponseWriter, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(data)
}
