package graphql

import (
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/mutation"
	"jobqueue/delivery/graphql/query"
	_interface "jobqueue/interface"
)

// rootResolver is the entry point for all GraphQL queries and mutations.
type rootResolver struct {
	// Embed the query and mutation resolvers
	query.JobQuery
	mutation.JobMutation
}

// NewRootResolver creates and returns a new rootResolver instance.
// It takes the JobService and a GeneralDataloader as dependencies
// and initializes the JobQuery and JobMutation resolvers.
func NewRootResolver(jobService _interface.JobService, dl *_dataloader.GeneralDataloader) *rootResolver {
	return &rootResolver{
		// Initialize JobQuery with its dependencies
		JobQuery: query.NewJobQuery(jobService, dl),
		// Initialize JobMutation with its dependencies
		JobMutation: mutation.NewJobMutation(jobService, dl),
	}
}