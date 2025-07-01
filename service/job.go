package service

import (
	"context"
	"fmt"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"log"
	"time"

	uuid "github.com/satori/go.uuid"
)

// jobService implements the _interface.JobService.
type jobService struct {
	jobRepo    _interface.JobRepository
	// jobProcessorCh is a channel to send jobs for asynchronous processing.
	// We'll wire this up to a worker pool in the next step.
	jobProcessorCh chan string
}

// NewJobService creates and returns a new JobService instance.
func NewJobService(repo _interface.JobRepository, processorCh chan string) _interface.JobService {
	return &jobService{
		jobRepo:        repo,
		jobProcessorCh: processorCh,
	}
}

// Enqueue creates a new job and sends it for processing.
func (s *jobService) Enqueue(ctx context.Context, taskName string, payload string, maxRetries int32) (string, error) {
	newJobID := uuid.NewV4().String() // Generate a new unique job ID
	now := time.Now()

	job := &entity.Job{
		ID:         newJobID,
		Task:       taskName,
		Payload:    payload,
		Status:     entity.JobStatusPending,
		Attempts:   0,
		MaxRetries: maxRetries,
		CreatedAt:  now,
	}

	if err := s.jobRepo.Save(ctx, job); err != nil {
		log.Printf("ERROR: Failed to enqueue job %s: %v", newJobID, err)
		return "", fmt.Errorf("failed to save job: %w", err)
	}

	log.Printf("INFO: Job %s (Task: %s) enqueued successfully. Status: %s", newJobID, taskName, job.Status)

	// Send job ID to the processor channel for asynchronous execution
	select {
	case s.jobProcessorCh <- newJobID:
		log.Printf("INFO: Job %s sent to processor channel.", newJobID)
	case <-ctx.Done():
		log.Printf("WARNING: Context cancelled while sending job %s to processor channel. Job might not be processed.", newJobID)
		return "", ctx.Err()
	default:
		// This case happens if the channel buffer is full, implying backpressure.
		// For now, we'll just log it. A more robust solution might involve retrying or a dedicated queue.
		log.Printf("WARNING: Processor channel full. Job %s could not be immediately sent for processing.", newJobID)
	}


	return newJobID, nil
}

// GetJobByID retrieves a specific job by its ID.
func (s *jobService) GetJobByID(ctx context.Context, id string) (*entity.Job, error) {
	job, err := s.jobRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("ERROR: Failed to retrieve job by ID %s: %v", id, err)
		return nil, fmt.Errorf("job %s not found: %w", id, err)
	}
	return job, nil
}

// GetAllJobs retrieves all registered jobs.
func (s *jobService) GetAllJobs(ctx context.Context) ([]*entity.Job, error) {
	jobs, err := s.jobRepo.FindAll(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to retrieve all jobs: %v", err)
		return nil, fmt.Errorf("failed to retrieve all jobs: %w", err)
	}
	return jobs, nil
}

// GetAllJobStatus retrieves aggregated statistics of all jobs by their status.
func (s *jobService) GetAllJobStatus(ctx context.Context) (*entity.JobStatusStatistics, error) {
	stats, err := s.jobRepo.GetStatusStatistics(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to retrieve job status statistics: %v", err)
		return nil, fmt.Errorf("failed to retrieve job status statistics: %w", err)
	}
	return stats, nil
}

// ProcessJob executes a single job, managing its state transitions and retries.
func (s *jobService) ProcessJob(ctx context.Context, jobID string) error {
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		log.Printf("ERROR: ProcessJob - Job %s not found: %v", jobID, err)
		return fmt.Errorf("job %s not found for processing: %w", jobID, err)
	}

	// Idempotency: Do not process if already completed or running
	if job.Status == entity.JobStatusCompleted || job.Status == entity.JobStatusRunning {
		log.Printf("INFO: ProcessJob - Job %s already in status %s. Skipping processing.", jobID, job.Status)
		return nil // Successfully skipped
	}
	// Check if already failed and retries exceeded
	if job.Status == entity.JobStatusFailed && job.Attempts >= job.MaxRetries {
		log.Printf("INFO: ProcessJob - Job %s failed and max retries (%d) exceeded. Skipping.", jobID, job.MaxRetries)
		return nil
	}

	// Transition to RUNNING
	originalStatus := job.Status
	job.Status = entity.JobStatusRunning
	now := time.Now()
	if job.StartedAt == nil { // Set StartedAt only once
		job.StartedAt = &now
	}
	job.Attempts++ // Increment attempt count
	job.Error = "" // Clear previous error
	if err := s.jobRepo.Update(ctx, job); err != nil {
		log.Printf("ERROR: ProcessJob - Failed to update job %s to RUNNING: %v", jobID, err)
		return fmt.Errorf("failed to update job status to RUNNING: %w", err)
	}
	log.Printf("INFO: ProcessJob - Job %s transitioned from %s to %s. Attempt %d/%d.",
		jobID, originalStatus, job.Status, job.Attempts, job.MaxRetries)

	var taskErr error
	// Simulate task execution
	switch job.Task {
	case "unstable-job":
		// Fails twice before succeeding
		if job.Attempts <= 2 {
			taskErr = fmt.Errorf("simulated failure %d for unstable-job %s", job.Attempts, jobID)
			log.Printf("WARN: ProcessJob - Unstable job %s failed on attempt %d.", jobID, job.Attempts)
		} else {
			log.Printf("INFO: ProcessJob - Unstable job %s succeeded on attempt %d.", jobID, job.Attempts)
		}
	default:
		// Default successful execution
		log.Printf("INFO: ProcessJob - Generic job %s (Task: %s) completed successfully.", jobID, job.Task)
	}

	// Handle task outcome
	if taskErr != nil {
		// Task failed, determine if retrying or truly failed
		if job.Attempts < job.MaxRetries {
			job.Status = entity.JobStatusRetrying
			job.Error = taskErr.Error()
			log.Printf("INFO: ProcessJob - Job %s failed, retrying (attempt %d/%d). Error: %v",
				jobID, job.Attempts, job.MaxRetries, taskErr)
		} else {
			job.Status = entity.JobStatusFailed
			job.CompletedAt = &now // Set completion time on final failure
			job.Error = taskErr.Error()
			log.Printf("ERROR: ProcessJob - Job %s failed permanently after %d attempts. Error: %v",
				jobID, job.Attempts, taskErr)
		}
	} else {
		// Task succeeded
		job.Status = entity.JobStatusCompleted
		job.CompletedAt = &now
		job.Error = "" // Clear any previous error
		log.Printf("INFO: ProcessJob - Job %s completed successfully.", jobID)
	}

	// Save the final/updated state of the job
	if err := s.jobRepo.Update(ctx, job); err != nil {
		log.Printf("CRITICAL ERROR: ProcessJob - Failed to save final state for job %s: %v", jobID, err)
		return fmt.Errorf("failed to save final job state for %s: %w", jobID, err)
	}

	return nil
}