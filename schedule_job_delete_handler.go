package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/vvakame/sdlog/aelog"
	"google.golang.org/api/googleapi"
	schedulerpb "google.golang.org/genproto/googleapis/cloud/scheduler/v1"
)

func (h *Handlers) ScheduleJobDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := aelog.WithHTTPRequest(r.Context(), r)
	if err := h.scheduleJobDeleteHandlerInternal(ctx, w, r); err != nil {
		// noop
	}
}

func (h *Handlers) scheduleJobDeleteHandlerInternal(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
	ctx = StartSpan(ctx, "scheduleJobDeleteHandler")
	defer EndSpan(ctx, err)

	jobName := r.FormValue("jobName") // `projects/PROJECT_ID/locations/LOCATION_ID/jobs/JOB_ID`. を期待している
	SetAttributesKV(ctx, map[string]interface{}{"jobNameParam": jobName})
	if err := h.scheduler.DeleteJob(ctx, &schedulerpb.DeleteJobRequest{
		Name: jobName,
	}); err != nil {
		fmt.Printf("failed Scheduler Job Delete:%s : %s\n", jobName, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	_, err = h.scheduler.GetJob(ctx, &schedulerpb.GetJobRequest{
		Name: jobName,
	})
	gerr := &googleapi.Error{}
	if errors.As(err, &gerr) && gerr.Code == http.StatusNotFound {
		// ちゃんと消えてる
	} else {
		fmt.Printf("Schedule Job Get %s", err)
	}

	time.Sleep(5 * time.Minute) // すやすや
	w.WriteHeader(http.StatusOK)
	return nil
}
