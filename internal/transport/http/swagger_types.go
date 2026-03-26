package http

// ErrorResponse is the generic error envelope used in Swagger schemas.
type ErrorResponse struct {
	Message string `json:"message"`
}
