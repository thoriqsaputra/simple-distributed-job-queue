package worker

import (
	"context"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"log"
	"sync"
	"time"
)

// JobProcessor handles the concurrent processing of jobs.
type JobProcessor struct {
	service        _interface.JobService
	jobIDsChan     chan string // Channel to receive job IDs for processing
	workerCount    int         // Number of concurrent workers
	retryDelay     time.Duration // Delay before retrying a job
	shutdown       chan struct{} // Channel to signal shutdown
	wg             sync.WaitGroup // WaitGroup to wait for all goroutines to finish
	processingJobs sync.Map // To track currently processing jobs (for idempotency check on the worker side)
}

// NewJobProcessor creates a new JobProcessor instance.
// jobIDsChan is the channel through which job IDs are sent by the JobService for processing.
func NewJobProcessor(service _interface.JobService, jobIDsChan chan string, workerCount int, retryDelay time.Duration) *JobProcessor {
	if workerCount <= 0 {
		workerCount = 5 // Default worker count if invalid value is provided
		log.Printf("WARNING: Invalid workerCount provided, defaulting to %d", workerCount)
	}
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second // Default retry delay
		log.Printf("WARNING: Invalid retryDelay provided, defaulting to %v", retryDelay)
	}

	return &JobProcessor{
		service:        service,
		jobIDsChan:     jobIDsChan,
		workerCount:    workerCount,
		retryDelay:     retryDelay,
		shutdown:       make(chan struct{}),
		processingJobs: sync.Map{},
	}
}

// Start launches the worker goroutines.
func (jp *JobProcessor) Start(ctx context.Context) {
	log.Printf("INFO: Starting JobProcessor with %d workers...", jp.workerCount)

	for i := 0; i < jp.workerCount; i++ {
		jp.wg.Add(1)
		go jp.worker(ctx, i+1)
	}
	log.Println("INFO: JobProcessor workers launched.")
}

// worker is a goroutine that processes jobs.
func (jp *JobProcessor) worker(ctx context.Context, workerID int) {
	defer jp.wg.Done()
	log.Printf("INFO: Worker %d started.", workerID)

	for {
		select {
		case jobID := <-jp.jobIDsChan:
			// Basic idempotency check within the worker itself to prevent processing the same job multiple times
			// if it somehow gets enqueued multiple times quickly before its status is updated.
			// The JobService.ProcessJob also has idempotency logic.
			if _, loaded := jp.processingJobs.LoadOrStore(jobID, true); loaded {
				log.Printf("INFO: Worker %d - Job %s is already being processed. Skipping.", workerID, jobID)
				continue
			}

			log.Printf("INFO: Worker %d - Received job %s for processing.", workerID, jobID)
			
			// Context for individual job processing (can be extended with timeouts if needed)
			jobCtx, cancel := context.WithCancel(ctx)
			err := jp.service.ProcessJob(jobCtx, jobID)
			cancel() // Release resources associated with jobCtx

			if err != nil {
				log.Printf("ERROR: Worker %d - Error processing job %s: %v", workerID, jobID, err)
			}

			// After processing, check job status for potential retries
			job, findErr := jp.service.GetJobByID(ctx, jobID)
			if findErr != nil {
				log.Printf("ERROR: Worker %d - Failed to retrieve job %s after processing to check status: %v", workerID, jobID, findErr)
			} else if job.Status == entity.JobStatusRetrying {
				log.Printf("INFO: Worker %d - Job %s needs retry. Re-queuing after %v delay.", workerID, jobID, jp.retryDelay)
				go func(id string) {
					time.Sleep(jp.retryDelay)
					select {
					case jp.jobIDsChan <- id:
						log.Printf("INFO: Worker %d - Job %s re-queued for retry.", workerID, id)
					case <-jp.shutdown: // Check for shutdown while sleeping
						log.Printf("INFO: Worker %d - Shutdown signal received while re-queuing job %s for retry. Aborting re-queue.", workerID, id)
					case <-ctx.Done(): // Check for main context cancellation
						log.Printf("INFO: Worker %d - Context cancelled while re-queuing job %s for retry. Aborting re-queue.", workerID, id)
					}
				}(jobID)
			}
			jp.processingJobs.Delete(jobID) // Mark job as no longer actively processing by this worker
		case <-jp.shutdown:
			log.Printf("INFO: Worker %d received shutdown signal. Exiting.", workerID)
			return
		case <-ctx.Done():
			log.Printf("INFO: Worker %d received context cancellation. Exiting.", workerID)
			return
		}
	}
}

// Stop signals all workers to gracefully shut down.
func (jp *JobProcessor) Stop() {
	log.Println("INFO: Signaling JobProcessor workers to stop...")
	close(jp.shutdown) // Close the shutdown channel to signal workers
	jp.wg.Wait()       // Wait for all workers to finish their current tasks
	log.Println("INFO: All JobProcessor workers stopped.")
}