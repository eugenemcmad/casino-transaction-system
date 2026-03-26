package app

import "time"

const (
	// Application settings
	DefaultShutdownTimeout = 5 * time.Second

	// Log messages
	MsgAPIInitialized      = "API App initialized"
	MsgStartingAPI         = "Starting API server"
	MsgHTTPServerError     = "HTTP server error"
	MsgShuttingDownAPI     = "Shutting down API server..."
	MsgClosingDBConnection = "Closing database connection..."
	MsgHTTPServerShutdown  = "HTTP server shutdown error"
	MsgProcessorInitialized = "Processor App initialized"
	MsgStartingProcessor   = "Starting Kafka Processor"
	MsgKafkaConsumerError  = "Kafka consumer stopped with error"
)
