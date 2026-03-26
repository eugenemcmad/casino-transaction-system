package repository

const (
	// Database driver name
	DriverPostgres = "postgres"

	// Log messages
	MsgInitializingPostgres = "Initializing Postgres connection"
	MsgDBConnectionSuccess  = "DB connection established successfully"
	MsgDBPingFailed         = "DB ping failed"
	MsgErrorOpeningDB       = "Error opening DB connection"
	MsgSavingToDB           = "Saving transaction to DB"
	MsgTransactionSaved     = "Transaction saved successfully"
	MsgFailedToInsert       = "Failed to insert transaction"
	MsgFetchingFromDB       = "Fetching transactions from DB"
	MsgFailedToQuery        = "Failed to query transactions"
	MsgFailedToScanRow      = "Failed to scan transaction row"
	MsgRowsError            = "Rows error after scanning"
	MsgFetchedCount         = "Fetched transactions count"
	MsgClosingPostgres      = "Closing Postgres connection"
	MsgFailedToCloseRows    = "Failed to close rows"
	MsgErrorClosingDB       = "Error closing DB connection"

	// Metrics
	MetricPostgresQueriesTotal       = "postgres_queries_total"
	MetricPostgresQueryDurationMs    = "postgres_query_duration_ms"
	MetricPostgresConnectionAttempts = "postgres_connection_attempts_total"
	// SQL Queries
	QueryGetTransactionsBase = `
		SELECT user_id, type, amount, timestamp, created_at
		FROM transactions
		WHERE 1=1
	`
	QueryOrderByTimestampDesc = " ORDER BY timestamp DESC NULLS LAST"
)
