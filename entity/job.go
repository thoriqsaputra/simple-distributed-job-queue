package entity

import (
	"time"
)

// JobStatus represents the current status of a job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "PENDING"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusCompleted JobStatus = "COMPLETED"
	JobStatusFailed    JobStatus = "FAILED"
	JobStatusRetrying  JobStatus = "RETRYING"
)

// Job represents a single unit of work in the system.
type Job struct {
	ID          string     `json:"id"`
	Task        string     `json:"task"`        // The type or name of the task (e.g., "unstable-job")
	Payload     string     `json:"payload"`     // Data associated with the job (e.g., JSON string)
	Status      JobStatus  `json:"status"`
	Attempts    int32      `json:"attempts"`    // Number of attempts made for this job
	MaxRetries  int32      `json:"maxRetries"`  // Maximum number of retries allowed
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt"`   // Pointer because it might be nil if not started yet
	CompletedAt *time.Time `json:"completedAt"` // Pointer because it might be nil if not completed/failed yet
	Error       string     `json:"error"`       // Stores the last error message if the job failed
}

// JobStatusStatistics holds the counts of jobs in various statuses.
type JobStatusStatistics struct {
	Total     int32 `json:"total"`
	Pending   int32 `json:"pending"`
	Running   int32 `json:"running"`
	Completed int32 `json:"completed"`
	Failed    int32 `json:"failed"`
	Retrying  int32 `json:"retrying"`
}