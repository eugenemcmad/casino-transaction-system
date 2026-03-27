package repository

import "errors"

var (
	ErrRepoNotInitialized = errors.New("repository is not initialized")
	ErrDBUnavailable      = errors.New("database is unavailable")
)
