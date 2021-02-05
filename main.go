package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/datastore"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/google/uuid"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

var ds *datastore.Client

func initTracer(projectID string) {
	// Create and register a OpenCensus Stackdriver Trace exporter.
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: projectID,
	})
	if err != nil {
		log.Fatal(err)
	}
	trace.RegisterExporter(exporter)

	// By default, traces will be sampled relatively rarely. To change the
	// sampling frequency for your entire program, call ApplyConfig. Use a
	// ProbabilitySampler to sample a subset of traces, or use AlwaysSample to
	// collect a trace on every run.
	//
	// Be careful about using trace.AlwaysSample in a production application
	// with significant traffic: a new trace will be started and exported for
	// every request.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}

func initClient(ctx context.Context, projectID string) error {
	var err error
	ds, err = datastore.NewClient(ctx, projectID)
	return err
}

func main() {
	ctx := context.Background()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	var err error
	if metadata.OnGCE() {
		projectID, err = metadata.ProjectID()
		if err != nil {
			panic(err)
		}
	}

	initTracer(projectID)

	if err := initClient(ctx, projectID); err != nil {
		panic(err)
	}
	als, err := NewAccessLogStore(ctx, ds)
	if err != nil {
		panic(err)
	}

	hc := &http.Client{
		Transport: &ochttp.Transport{
			// Use Google Cloud propagation format.
			Propagation: &propagation.HTTPFormat{},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, err := als.Insert(ctx, &AccessLog{
			ID: uuid.New().String(),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
			return
		}

		var results []string
		urls := []string{"https://cloudrun-helloworld-d5aduuftyq-an.a.run.app", "https://cloudrun-otel-d5aduuftyq-an.a.run.app/hello"}
		for _, u := range urls {
			req, _ := http.NewRequest("GET", u, nil)
			req = req.WithContext(r.Context())

			resp, err := hc.Do(req)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err.Error())
				return
			}
			defer resp.Body.Close()
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
	})
	http.Handle("/hello", handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)

	httpHandler := &ochttp.Handler{
		// Use the Google Cloud propagation format.
		Propagation: &propagation.HTTPFormat{},
	}
	if err := http.ListenAndServe(":"+port, httpHandler); err != nil {
		log.Fatal(err)
	}
}
