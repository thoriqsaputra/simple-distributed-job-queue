package main

import (
	"context"
	"fmt"
	"jobqueue/config"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql"
	"jobqueue/pkg/handler"
	"jobqueue/pkg/server"
	"jobqueue/pkg/worker"
	inmemrepo "jobqueue/repository/inmem"
	"jobqueue/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	_graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Number of concurrent job processing workers
	WorkerCount = 10
	// Delay for retrying a failed job
	RetryDelay = 5 * time.Second
	// Buffer size for the job queue channel
	JobChannelBufferSize = 100
)

func main() {
	setupLogger()
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// =========================================================================
	// 1. Initialize In-Memory Repository
	// =========================================================================
	jobRepository := inmemrepo.NewJobRepository()
	log.Println("INFO: Initialized Job Repository (In-Memory).")

	// =========================================================================
	// 2. Initialize Job Processor Channel
	// This channel acts as the queue for jobs to be processed.
	// =========================================================================
	jobProcessorCh := make(chan string, JobChannelBufferSize)
	log.Printf("INFO: Initialized Job Processor Channel with buffer size %d.", JobChannelBufferSize)

	// =========================================================================
	// 3. Initialize Job Service (Core Business Logic)
	// It depends on the repository and the job processor channel.
	// =========================================================================
	jobService := service.NewJobService(jobRepository, jobProcessorCh)
	log.Println("INFO: Initialized Job Service.")

	// =========================================================================
	// 4. Initialize Dataloader
	// It depends on the repository for batch fetching.
	// =========================================================================
	dataloader := _dataloader.NewGeneralDataloader(jobRepository)
	log.Println("INFO: Initialized Dataloader.")

	// =========================================================================
	// 5. Initialize GraphQL Root Resolver
	// =========================================================================
	rootResolver := graphql.NewRootResolver(jobService, dataloader)
	log.Println("INFO: Initialized GraphQL Root Resolver.")

	// =========================================================================
	// 6. Initialize GraphQL Schema
	// The schema object is created from the schema string and the root resolver.
	// =========================================================================
	// Read schema strings from files
	querySchema, err := getQuerySchema()
	if err != nil {
		log.Fatalf("FATAL: Failed to load query schema: %v", err)
	}
	mutationSchema, err := getMutationSchema()
	if err != nil {
		log.Fatalf("FATAL: Failed to load mutation schema: %v", err)
	}
	typeSchema, err := getTypeSchema()
	if err != nil {
		log.Fatalf("FATAL: Failed to load type schema: %v", err)
	}

	schemaString := `
		schema {
			query: Query
			mutation: Mutation
		}
		` + querySchema + `
		` + mutationSchema + `
		` + typeSchema

	opts := make([]_graphql.SchemaOpt, 0)
	opts = append(opts, _graphql.SubscribeResolverTimeout(10*time.Second))
	graphqlSchema := _graphql.MustParseSchema(schemaString, rootResolver, opts...)
	log.Println("INFO: Parsed GraphQL Schema.")

	// =========================================================================
	// 7. Initialize Job Processor (Worker Pool)
	// =========================================================================
	jobProcessor := worker.NewJobProcessor(jobService, jobProcessorCh, WorkerCount, RetryDelay)

	// =========================================================================
	// 8. Setup Echo HTTP Server
	// =========================================================================
	e := server.New(config.Data.Server) // Assuming this initializes the echo instance

	// Add middleware chain
	e.Echo.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${remote_ip} ${time_rfc3339_nano} \"${method} ${path}\" ${status} ${bytes_out} \"${referer}\" \"${user_agent}\"\n",
	}))
	e.Echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.OPTIONS},
	}))

	// Register the dataloader middleware before the GraphQL handler
	e.Echo.Use(dataloader.EchoMiddelware)

	// Register GraphQL handler for POST and GET
	e.Echo.POST("/graphql", handler.GraphQLHandler(&relay.Handler{Schema: graphqlSchema}))
	e.Echo.GET("/graphql", handler.GraphQLHandler(&relay.Handler{Schema: graphqlSchema}))

	// Register GraphiQL handler
	e.Echo.GET("/graphiql", handler.GraphiQLHandler)
	log.Println("INFO: Echo server routes registered.")

	// =========================================================================
	// 9. Start Workers and Server
	// =========================================================================
	// Create a context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start the job processor workers in a goroutine
	jobProcessor.Start(ctx)
	log.Println("INFO: Job Processor workers started.")

	// Start the Echo server in a goroutine
	go func() {
		if err := e.Start(); err != nil && err != http.ErrServerClosed { // Corrected constant
			log.Fatalf("FATAL: Server startup failed: %v", err)
		}
	}()
	log.Println("INFO: Echo server started.")

	// =========================================================================
	// 10. Graceful Shutdown
	// =========================================================================
	<-ctx.Done()
	log.Println("INFO: Shutdown signal received. Starting graceful shutdown...")

	// Gracefully shut down the Echo server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("ERROR: Echo server shutdown failed: %v", err)
	}

	// Stop the job processor workers
	jobProcessor.Stop()
	log.Println("INFO: Job Processor workers stopped.")

	log.Println("INFO: Application shut down gracefully.")
}

// getQuerySchema reads the GraphQL query schema from its file.
func getQuerySchema() (string, error) {
	filePath := "delivery/graphql/schema/query.graphql"
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read query schema file %s: %w", filePath, err)
	}
	return string(content), nil
}

// getMutationSchema reads the GraphQL mutation schema from its file.
func getMutationSchema() (string, error) {
	filePath := "delivery/graphql/schema/mutation.graphql"
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read mutation schema file %s: %w", filePath, err)
	}
	return string(content), nil
}

// getTypeSchema reads the GraphQL type schema from its file.
func getTypeSchema() (string, error) {
	filePath := "delivery/graphql/schema/type/job.graphql" // This is where Job and JobStatus are defined
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read type schema file %s: %w", filePath, err)
	}
	return string(content), nil
}

// setupLogger initializes Zap logger as a global logger.
func setupLogger() {
	configLogger := zap.NewDevelopmentConfig()
	configLogger.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	configLogger.DisableStacktrace = true
	logger, _ := configLogger.Build()
	zap.ReplaceGlobals(logger)
}