package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

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

	initTracer(projectID)

	if err := initClient(ctx, projectID); err != nil {
		panic(err)
	}
	als, err := NewAccessLogStore(ctx, ds)
	if err != nil {
		panic(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, err := als.Insert(ctx, &AccessLog{
			ID: uuid.New().String(),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err.Error())
		}
		_, _ = io.WriteString(w, "Hello, world!\n")
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
