package common

import (
	"log"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/hibiken/asynq"
)

// client is the global asyncq client instance
var client *asynq.Client

// InitAsyncq initializes the asyncq client with configuration
// It creates a new client and starts the server in a goroutine
func InitAsyncq(config *config.Config) error {
	cfg := config.Asynq
	client = asynq.NewClient(asynq.RedisClientOpt{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
	go run(cfg)
	return nil
}

// GetAsyncqClient returns the global asyncq client instance
func GetAsyncqClient() *asynq.Client {
	return client
}

// handleFunc stores registered task handlers
var handleFunc = map[string]asynq.HandlerFunc{}

// RegisterHandlerFunc registers a handler function for a specific task type
func RegisterHandlerFunc(taskType string, handlerFunc asynq.HandlerFunc) {
	handleFunc[taskType] = handlerFunc
}

// run starts the asyncq server with the given configuration
// It creates a new server, sets up handlers, and runs the server
func run(config *config.AsynqConfig) {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:         config.Addr,
			Username:     config.Username,
			Password:     config.Password,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		},
		asynq.Config{
			Concurrency: config.Concurrency,
			Queues: map[string]int{
				"critical": 6, // Highest priority queue
				"default":  3, // Default priority queue
				"low":      1, // Lowest priority queue
			},
		},
	)

	// Create a new mux and register all handlers
	mux := asynq.NewServeMux()
	for typ, handler := range handleFunc {
		mux.HandleFunc(typ, handler)
	}

	// Start the server
	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
