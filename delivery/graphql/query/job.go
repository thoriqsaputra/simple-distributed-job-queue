package query

import (
	"context"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/resolver"
	_interface "jobqueue/interface"
	"log"
)

// JobQuery holds the dependencies for job-related GraphQL queries.
type JobQuery struct {
	jobService _interface.JobService
	dataloader *_dataloader.GeneralDataloader
}

// NewJobQuery creates and returns a new JobQuery instance.
func NewJobQuery(jobService _interface.JobService, dataloader *_dataloader.GeneralDataloader) JobQuery {
	return JobQuery{
		jobService: jobService,
		dataloader: dataloader,
	}
}

// Jobs resolves the 'Jobs' query, returning a list of all jobs.
func (q JobQuery) Jobs(ctx context.Context) ([]*resolver.JobResolver, error) {
	jobs, err := q.jobService.GetAllJobs(ctx)
	if err != nil {
		log.Printf("ERROR: GraphQL Query Jobs - Failed to get all jobs: %v", err)
		return nil, err
	}

	var resolvers []*resolver.JobResolver
	for _, job := range jobs {
		resolvers = append(resolvers, &resolver.JobResolver{Data: job, JobService: q.jobService})
	}
	log.Printf("INFO: GraphQL Query Jobs - Returned %d jobs.", len(resolvers))
	return resolvers, nil
}

// Job resolves the 'Job' query, returning a single job by its ID.
func (q JobQuery) Job(ctx context.Context, args struct {
	ID string
}) (*resolver.JobResolver, error) {
	job, err := q.jobService.GetJobByID(ctx, args.ID)
	if err != nil {
		log.Printf("ERROR: GraphQL Query Job by ID %s - Failed to get job: %v", args.ID, err)
		return nil, err
	}
	log.Printf("INFO: GraphQL Query Job by ID %s - Returned job.", args.ID)
	return &resolver.JobResolver{Data: job, JobService: q.jobService}, nil
}

// JobStatus resolves the 'JobStatus' query, returning aggregated job status statistics.
func (q JobQuery) JobStatus(ctx context.Context) (*resolver.JobStatusResolver, error) { // Return pointer here
	stats, err := q.jobService.GetAllJobStatus(ctx)
	if err != nil {
		log.Printf("ERROR: GraphQL Query JobStatus - Failed to get job status statistics: %v", err)
		return nil, err
	}
	log.Println("INFO: GraphQL Query JobStatus - Returned statistics.")
	return &resolver.JobStatusResolver{Data: stats, JobService: q.jobService}, nil
}