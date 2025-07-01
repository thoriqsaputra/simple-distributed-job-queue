package mutation

import (
	"context"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/resolver"
	_interface "jobqueue/interface"
	"log"
)

// NewJobInput represents the input structure for creating a new job via GraphQL.
// This struct should match the `NewJobInput` in your GraphQL schema.
type NewJobInput struct {
	Task       string  `json:"task"`
	Payload    *string `json:"payload"`
	MaxRetries int32   `json:"maxRetries"`
}

// JobMutation holds the dependencies for job-related GraphQL mutations.
type JobMutation struct {
	jobService _interface.JobService
	dataloader *_dataloader.GeneralDataloader
}

// NewJobMutation to create new instance
func NewJobMutation(jobService _interface.JobService, dataloader *_dataloader.GeneralDataloader) JobMutation {
	return JobMutation{
		jobService: jobService,
		dataloader: dataloader,
	}
}

// SimultaneousCreateJob resolves the 'SimultaneousCreateJob' mutation,
// allowing the creation of multiple jobs in a single request.
func (m JobMutation) SimultaneousCreateJob(ctx context.Context, args struct {
	Jobs []*NewJobInput
}) ([]*resolver.JobResolver, error) {
	var createdJobResolvers []*resolver.JobResolver
	
	if len(args.Jobs) == 0 {
		log.Println("WARNING: SimultaneousCreateJob called with no job inputs.")
		return []*resolver.JobResolver{}, nil
	}

	for _, input := range args.Jobs {
		// No need to check for nil on MaxRetries, as the GraphQL library will
		// automatically provide the default value (3) if not supplied by the client.
		maxRetries := input.MaxRetries

		// Handle nullable payload
		payload := ""
		if input.Payload != nil {
			payload = *input.Payload
		}

		jobID, err := m.jobService.Enqueue(ctx, input.Task, payload, maxRetries)
		if err != nil {
			log.Printf("ERROR: GraphQL Mutation SimultaneousCreateJob - Failed to enqueue job (Task: %s): %v", input.Task, err)
			// Return the error and stop processing the batch
			return nil, err
		}

		// Retrieve the newly created job to return its full details
		createdJob, err := m.jobService.GetJobByID(ctx, jobID)
		if err != nil {
			log.Printf("ERROR: GraphQL Mutation SimultaneousCreateJob - Failed to retrieve newly created job %s: %v", jobID, err)
			return nil, err
		}

		createdJobResolvers = append(createdJobResolvers, &resolver.JobResolver{
			Data:       createdJob,
			JobService: m.jobService,
			Dataloader: m.dataloader,
		})
		log.Printf("INFO: GraphQL Mutation SimultaneousCreateJob - Enqueued job %s (Task: %s).", jobID, input.Task)
	}

	log.Printf("INFO: GraphQL Mutation SimultaneousCreateJob - Successfully enqueued %d jobs.", len(createdJobResolvers))
	return createdJobResolvers, nil
}