package server

import (
	"context" // Import context for the Shutdown method
	"fmt"
	"os"

	echo "github.com/labstack/echo/v4"
)

var (
	config Config
)

// Config hold configuration data
type Config struct {
	Port int
}

// Echo hold echo wrapper
type Echo struct {
	Echo *echo.Echo
}

// Start hold echo start wrapper
// Changed receiver to a pointer (*Echo) for idiomatic Go.
func (e *Echo) Start() error {
	var host string
	env := os.Getenv("NODE_ENV")
	if env != "staging" && env != "production" {
		host = "localhost"
	}
	// This method will return `echo.ErrServerClosed` during graceful shutdown.
	return e.Echo.Start(fmt.Sprintf("%s:%d", host, config.Port))
}

// Shutdown gracefully shuts down the underlying Echo server.
// This method is required for the graceful shutdown logic in main.go.
func (e *Echo) Shutdown(ctx context.Context) error {
	return e.Echo.Shutdown(ctx)
}

// New generate new echo server
func New(d Config) *Echo {
	// Store the config
	config = d

	return &Echo{echo.New()}
}