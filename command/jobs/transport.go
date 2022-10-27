package jobs

import (
	"context"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/handler"
)

type Job struct {
	ctx      context.Context
	handler  *handler.Endpoint
	job      *config.Job
	settings *config.Settings
}

var jobs = make(map[string]*Job)

func AddJob(ctx context.Context, j *config.Job, h *handler.Endpoint, settings *config.Settings) {
	jobs[j.Name] = &Job{
		ctx:      ctx,
		handler:  h,
		job:      j,
		settings: settings,
	}
}

func GetJobs() map[string]*Job {
	return jobs
}

func (j *Job) GetCtx() context.Context {
	return j.ctx
}

func (j *Job) GetHandler() *handler.Endpoint {
	return j.handler
}

func (j *Job) GetJob() *config.Job {
	return j.job
}

func (j *Job) GetSettings() *config.Settings {
	return j.settings
}
