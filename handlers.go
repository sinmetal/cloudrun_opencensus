package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"

	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/vvakame/sdlog/aelog"
	"go.opencensus.io/plugin/ochttp"
)

type Handlers struct {
	als *AccessLogStore
}

func (h *Handlers) HelloHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.helloHandlerInternal(r.Context(), w, r); err != nil {
		// noop
	}
}

func (h *Handlers) helloHandlerInternal(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
	ctx = StartSpan(ctx, "helloHandler")
	defer EndSpan(ctx, err)

	id := uuid.New().String()
	_, err = h.als.Insert(ctx, &AccessLog{
		ID: id,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}
	msg := r.FormValue("message")
	aelog.Infof(ctx, "AccessLogID : %s", id)
	aelog.Infof(ctx, "Message : %s", msg)
	SetAttributesKV(ctx, map[string]interface{}{
		"AccessLogID": id,
		"Message":     msg,
	})

	hc := &http.Client{
		Transport: &ochttp.Transport{
			// Use Google Cloud propagation format.
			Propagation: &propagation.HTTPFormat{},
		},
	}

	var results []string
	urls := []string{"https://cloudrun-helloworld-d5aduuftyq-an.a.run.app", "https://cloudrun-otel-d5aduuftyq-an.a.run.app/hello"}
	for _, u := range urls {
		req, _ := http.NewRequest("GET", u, nil)
		req = req.WithContext(ctx)

		resp, err := hc.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Println(err.Error())
			}
		}()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}
		results = append(results, string(b))
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Println(err.Error())
	}
	return nil
}
