package http

const (
	// Error messages
	MsgInvalidUserID               = "invalid user_id"
	MsgFailedToGetTransactions     = "failed to get transactions"
	MsgInvalidTransactionTypeInReq = "invalid transaction type" // For GET query param

	// Metrics
	MetricAPIRequestsTotal     = "api_requests_total"
	MetricAPIRequestDurationMs = "api_request_duration_ms"

	// HTTP Headers
	HeaderContentType   = "Content-Type"
	MimeApplicationJSON = "application/json"
)
