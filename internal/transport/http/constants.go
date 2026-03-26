package http

const (
	// Error messages
	MsgInvalidRequestBody          = "invalid request body"
	MsgInvalidTransactionType      = "invalid transaction type: use 'bet' or 'win'"
	MsgUserIDAmountMustBePositive  = "user_id and amount must be positive"
	MsgUserIDRequired              = "user_id is required"
	MsgInvalidUserID               = "invalid user_id"
	MsgFailedToRegisterTransaction = "failed to register transaction"
	MsgFailedToGetTransactions     = "failed to get transactions"
	MsgInvalidTransactionTypeInReq = "invalid transaction type" // For GET query param
	MsgMissingZeroTimestamp        = "received transaction with missing/zero timestamp"
	MsgTransactionProcessed        = "Transaction registered via API"

	// Response messages
	StatusRegistered = "registered"

	// HTTP Headers
	HeaderContentType   = "Content-Type"
	MimeApplicationJSON = "application/json"
)
