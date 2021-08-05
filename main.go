package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/datastore"
	scheduler "cloud.google.com/go/scheduler/apiv1"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
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

	schedulerClient, err := scheduler.NewCloudSchedulerClient(ctx)
	if err != nil {
		panic(err)
	}

	handlers := &Handlers{
		als:       als,
		scheduler: schedulerClient,
	}

	http.HandleFunc("/hello", handlers.HelloHandler)
	http.HandleFunc("/deleteSchedulerJob", handlers.ScheduleJobDeleteHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Listening on port %s", port)

	httpHandler := &ochttp.Handler{
		// Use the Google Cloud propagation format.
		Propagation: &propagation.HTTPFormat{},
	}
	if err := http.ListenAndServe(":"+port, httpHandler); err != nil {
		log.Fatal(err)
	}
}
