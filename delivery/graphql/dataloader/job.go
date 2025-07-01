package _dataloader

import (
	"context"
	"fmt"
	"jobqueue/entity" // Import entity to use Job
	"log"

	"github.com/graph-gophers/dataloader/v6"
)

// JobBatchFunc is the batch function for loading Jobs by ID.
// It receives a slice of keys (job IDs) and is expected to return
// a slice of results (jobs or errors) in the same order as the keys.
func (g *GeneralDataloader) JobBatchFunc(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	jobIDs := make([]string, len(keys))

	// Extract job IDs from dataloader keys
	for i, key := range keys {
		jobIDs[i] = key.String()
	}

	log.Printf("INFO: Dataloader BatchFunc - Batching %d job IDs: %v", len(jobIDs), jobIDs)

	// Use the batch-fetching method from the repository
	jobs, err := g.jobRepo.FindManyByIDs(ctx, jobIDs)
	if err != nil {
		// If a global error occurs, set error for all results
		for i := range keys {
			results[i] = &dataloader.Result{Error: fmt.Errorf("failed to fetch jobs in batch: %w", err)}
		}
		log.Printf("ERROR: Dataloader BatchFunc - Failed to fetch jobs in batch: %v", err)
		return results
	}

	// Map the fetched jobs back to the original request order
	// This is crucial as dataloader expects results to match the order of keys.
	jobMap := make(map[string]*entity.Job)
	for _, job := range jobs {
		if job != nil { // Ensure nil jobs from FindManyByIDs (for not-found) are not added to map
			jobMap[job.ID] = job
		}
	}

	for i, id := range jobIDs {
		if job, ok := jobMap[id]; ok {
			results[i] = &dataloader.Result{Data: job}
		} else {
			// If a job was not found by FindManyByIDs (returned nil for that ID),
			// set a nil result. The dataloader will then return nil to the resolver.
			results[i] = &dataloader.Result{Data: nil}
			log.Printf("INFO: Dataloader BatchFunc - Job %s not found in batch result.", id)
		}
	}

	log.Printf("INFO: Dataloader BatchFunc - Successfully processed batch for %d job IDs.", len(jobIDs))
	return results
}