package _interface

import (
	"context"
	"jobqueue/entity"
)

type JobService interface {
	// Enqueue creates and registers a new job with the specified task, payload, and max retries.
	Enqueue(ctx context.Context, taskName string, payload string, maxRetries int32) (string, error)

	// GetJobByID retrieves a specific job by its ID.
	GetJobByID(ctx context.Context, id string) (*entity.Job, error)

	// GetAllJobs retrieves all registered jobs.
	GetAllJobs(ctx context.Context) ([]*entity.Job, error)

	// GetAllJobStatus retrieves aggregated statistics of all jobs by their status.
	GetAllJobStatus(ctx context.Context) (*entity.JobStatusStatistics, error)

	// ProcessJob executes a single job, managing its state transitions and retries.
	// This method is intended to be called by a worker/consumer.
	ProcessJob(ctx context.Context, jobID string) error
}

type JobRepository interface {
	Save(ctx context.Context, job *entity.Job) error
	FindByID(ctx context.Context, id string) (*entity.Job, error)
	FindAll(ctx context.Context) ([]*entity.Job, error)
	// New methods
	Update(ctx context.Context, job *entity.Job) error
	GetStatusStatistics(ctx context.Context) (*entity.JobStatusStatistics, error)
	FindManyByIDs(ctx context.Context, ids []string) ([]*entity.Job, error)
}
