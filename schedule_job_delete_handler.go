package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vvakame/sdlog/aelog"
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
		Name: fmt.Sprintf(jobName),
	}); err != nil {
		fmt.Printf("failed Scheduler Job Delete:%s : %s\n", jobName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}
