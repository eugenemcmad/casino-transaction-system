package repository

const (
	// Database driver name
	DriverPostgres = "postgres"

	// Log messages
	MsgInitializingPostgres  = "Initializing Postgres connection"
	MsgDBConnectionSuccess   = "DB connection established successfully"
	MsgDBPingFailed          = "DB ping failed"
	MsgErrorOpeningDB        = "Error opening DB connection"
	MsgSavingToDB            = "Saving transaction to DB"
	MsgTransactionSaved      = "Transaction saved successfully"
	MsgFailedToInsert        = "Failed to insert transaction"
	MsgFetchingFromDB        = "Fetching transactions from DB"
	MsgFailedToQuery         = "Failed to query transactions by user ID"
	MsgFailedToScanRow       = "Failed to scan transaction row"
	MsgRowsError             = "Rows error after scanning"
	MsgFetchedCount          = "Fetched transactions count"
	MsgClosingPostgres       = "Closing Postgres connection"

	// SQL Queries
	QueryInsertTransaction = `
		INSERT INTO transactions (user_id, type, amount, timestamp)
		VALUES ($1, $2, $3, $4)
	`
	QueryGetTransactionsBase = `
		SELECT user_id, type, amount, timestamp, created_at
		FROM transactions
		WHERE user_id = $1
	`
	QueryOrderByTimestampDesc = " ORDER BY timestamp DESC NULLS LAST"
)
