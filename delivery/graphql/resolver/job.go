package resolver

import (
	"context" // Important for resolver methods
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"time" // For formatting time fields

	_graphql "github.com/graph-gophers/graphql-go"
)

// JobResolver resolves fields for the 'Job' GraphQL type.
type JobResolver struct {
	Data       *entity.Job
	JobService _interface.JobService
	Dataloader *_dataloader.GeneralDataloader
}

// JobStatusResolver resolves fields for the 'JobStatus' GraphQL type (which maps to entity.JobStatusStatistics).
type JobStatusResolver struct {
	Data       *entity.JobStatusStatistics
	JobService _interface.JobService
	Dataloader *_dataloader.GeneralDataloader
}

// =========================================================================
// JobResolver Field Implementations
// =========================================================================

// ID resolves the 'id' field of a Job.
func (r *JobResolver) ID(ctx context.Context) _graphql.ID {
	return _graphql.ID(r.Data.ID)
}

// Task resolves the 'task' field of a Job.
func (r *JobResolver) Task(ctx context.Context) string {
	return r.Data.Task
}

// Payload resolves the 'payload' field of a Job.
func (r *JobResolver) Payload(ctx context.Context) *string { // Returns *string because it's nullable in GraphQL
	if r.Data.Payload == "" {
		return nil
	}
	return &r.Data.Payload
}

// Status resolves the 'status' field of a Job.
func (r *JobResolver) Status(ctx context.Context) string {
	return string(r.Data.Status) // Convert entity.JobStatus to string
}

// Attempts resolves the 'attempts' field of a Job.
func (r *JobResolver) Attempts(ctx context.Context) int32 {
	return r.Data.Attempts
}

// MaxRetries resolves the 'maxRetries' field of a Job.
func (r *JobResolver) MaxRetries(ctx context.Context) int32 {
	return r.Data.MaxRetries
}

// CreatedAt resolves the 'createdAt' field of a Job.
func (r *JobResolver) CreatedAt(ctx context.Context) string {
	return r.Data.CreatedAt.Format(time.RFC3339) // Format time to a standard string
}

// StartedAt resolves the 'startedAt' field of a Job.
func (r *JobResolver) StartedAt(ctx context.Context) *string {
	if r.Data.StartedAt == nil {
		return nil
	}
	t := r.Data.StartedAt.Format(time.RFC3339)
	return &t
}

// CompletedAt resolves the 'completedAt' field of a Job.
func (r *JobResolver) CompletedAt(ctx context.Context) *string {
	if r.Data.CompletedAt == nil {
		return nil
	}
	t := r.Data.CompletedAt.Format(time.RFC3339)
	return &t
}

// Error resolves the 'error' field of a Job.
func (r *JobResolver) Error(ctx context.Context) *string {
	if r.Data.Error == "" {
		return nil
	}
	return &r.Data.Error
}

// =========================================================================
// JobStatusResolver Field Implementations
// =========================================================================

// Total resolves the 'total' field of JobStatus.
func (r *JobStatusResolver) Total(ctx context.Context) int32 {
	return r.Data.Total
}

// Pending resolves the 'pending' field of JobStatus.
func (r *JobStatusResolver) Pending(ctx context.Context) int32 {
	return r.Data.Pending
}

// Running resolves the 'running' field of JobStatus.
func (r *JobStatusResolver) Running(ctx context.Context) int32 {
	return r.Data.Running
}

// Failed resolves the 'failed' field of JobStatus.
func (r *JobStatusResolver) Failed(ctx context.Context) int32 {
	return r.Data.Failed
}

// Completed resolves the 'completed' field of JobStatus.
func (r *JobStatusResolver) Completed(ctx context.Context) int32 {
	return r.Data.Completed
}

// Retrying resolves the 'retrying' field of JobStatus.
func (r *JobStatusResolver) Retrying(ctx context.Context) int32 {
	return r.Data.Retrying
}
