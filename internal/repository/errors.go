package repository

import "errors"

// Sentinel errors for PostgreSQL repository initialization and connectivity.
var (
	ErrRepoNotInitialized = errors.New("repository is not initialized")
	ErrDBUnavailable      = errors.New("database is unavailable")
)
