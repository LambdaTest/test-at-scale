package errs

import (
	"fmt"
)

// Err represents structure of a custom error
type Err struct {
	Code    string
	Message string
	URL     string
}

func (e Err) Error() string {
	return fmt.Sprintf("%s : %s ", e.Code, e.Message)
}

// Error represents a json-encoded API error.
type Error struct {
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// New returns a new error message.
func New(text string) error {
	return &Error{Message: text}
}

// ErrInvalidPayload returns an error when the  nucleus payload is invalid.
func ErrInvalidPayload(errMsg string) error {
	return New(errMsg)
}

// ErrSecretNotFound represents the error when a secret is not found in map.
func ErrSecretNotFound(secret string) error {
	return New(fmt.Sprintf("secret with name %s not found", secret))
}

var (
	// ErrParseVariableName represents the error when unable to parse a
	// variable name within a substitution.
	ErrParseVariableName = New("unable to parse variable name")
	// ErrSecretRegexMatch represents the error when a regex does not match.
	ErrSecretRegexMatch = New("secret regex match failed")
	// ErrNotFound return when azure blob is not found.
	ErrNotFound = New("blob not found")
	// ErrSASToken returns when sas token is not found.
	ErrSASToken = New("azure client requires SAS Token")
	// ErrAzureCredentials is returned when the azure credentials are invalid.
	ErrAzureCredentials = New("azure client requires credentials")
	// ErrApiStatus is returned when the api status is not 200.
	ErrApiStatus = New("non OK status")
	// ErrInvalidLoggerInstance is returned when logger instance is not supported.
	ErrInvalidLoggerInstance = New("Invalid logger instance")
	// ErrUnsupportedGitProvider is returned when try to integrate unsupported provider repo
	ErrUnsupportedGitProvider = New("unsupported gitprovider")
	// ErrGitDiffNotFound is returned when basecommit is null or git provider returns empty diff
	ErrGitDiffNotFound = New("diff not found")
	// GenericErrRemark returns a generic error message for user facing errors.
	GenericErrRemark = New("Unexpected error")
)
