package inmemrepo

import (
	"context"
	"errors"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"sync"
)

// jobRepository implements the _interface.JobRepository using an in-memory map.
type jobRepository struct {
	mu      sync.RWMutex
	inMemDb map[string]*entity.Job
}

// NewJobRepository creates and returns a new in-memory job repository.
func NewJobRepository() _interface.JobRepository {
	return &jobRepository{
		inMemDb: make(map[string]*entity.Job),
	}
}

// Save saves a new job or updates an existing job in the in-memory database.
// This method handles both creation and full updates for a job.
func (t *jobRepository) Save(ctx context.Context, job *entity.Job) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Ensure job ID is not empty
	if job.ID == "" {
		return errors.New("job ID cannot be empty for save/update operation")
	}

	t.inMemDb[job.ID] = job
	return nil
}

// FindByID retrieves a job by its ID from the in-memory database.
func (t *jobRepository) FindByID(ctx context.Context, id string) (*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	job, exists := t.inMemDb[id]
	if !exists {
		return nil, errors.New("job not found")
	}
	// Return a copy to prevent external modification without using Save/Update
	copiedJob := *job
	return &copiedJob, nil
}

// FindAll retrieves all jobs from the in-memory database.
func (t *jobRepository) FindAll(ctx context.Context) ([]*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var jobs []*entity.Job
	for _, job := range t.inMemDb {
		// Return copies to prevent external modification without using Save/Update
		copiedJob := *job
		jobs = append(jobs, &copiedJob)
	}
	return jobs, nil
}

// Update updates an existing job in the in-memory database.
// It's explicitly added for semantic clarity, though Save could perform similar function.
func (t *jobRepository) Update(ctx context.Context, job *entity.Job) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if job.ID == "" {
		return errors.New("job ID cannot be empty for update operation")
	}

	// Check if the job exists before updating
	if _, exists := t.inMemDb[job.ID]; !exists {
		return errors.New("job not found for update")
	}

	t.inMemDb[job.ID] = job
	return nil
}

// GetStatusStatistics calculates and returns the count of jobs for each status.
func (t *jobRepository) GetStatusStatistics(ctx context.Context) (*entity.JobStatusStatistics, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := &entity.JobStatusStatistics{
		Total: 0,
		Pending:   0,
		Running:   0,
		Completed: 0,
		Failed:    0,
		Retrying:  0,
	}

	for _, job := range t.inMemDb {
		println("Processing job ID:", job.ID, "with status:", job.Status) // Debugging output
		stats.Total++
		switch job.Status {
		case entity.JobStatusPending:
			stats.Pending++
		case entity.JobStatusRunning:
			stats.Running++
		case entity.JobStatusCompleted:
			stats.Completed++
		case entity.JobStatusFailed:
			stats.Failed++
		case entity.JobStatusRetrying:
			stats.Retrying++
		}
	}
	return stats, nil
}

// FindManyByIDs retrieves multiple jobs by their IDs from the in-memory database.
// This is an efficient batch-loading method for the Dataloader.
func (t *jobRepository) FindManyByIDs(ctx context.Context, ids []string) ([]*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create a map for quick lookups
	resultsMap := make(map[string]*entity.Job)
	for _, job := range t.inMemDb {
		resultsMap[job.ID] = job
	}

	// Build the result slice in the same order as the input IDs, with nil for not-found items.
	resultsOrdered := make([]*entity.Job, len(ids))
	for i, id := range ids {
		if job, ok := resultsMap[id]; ok {
			copiedJob := *job // Return a copy to prevent external modification
			resultsOrdered[i] = &copiedJob
		} else {
			resultsOrdered[i] = nil // Not found, Dataloader expects nil
		}
	}

	return resultsOrdered, nil
}