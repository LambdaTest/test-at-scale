package errs

import (
	"testing"
)

func TestError_Error(t *testing.T) {
	e := New("A secret message")
	got := e.Error()
	want := "A secret message"
	if got != want {
		t.Errorf("Received: %v, Expected: %v", got, want)
	}
}

func TestErr_Error(t *testing.T) {
	e := &Err{
		Code:    "fmt.Print(error)",
		Message: "This is the message",
	}
	got := e.Error()
	want := "fmt.Print(error) : This is the message "
	if got != want {
		t.Errorf("Received: %v, Expected: %v", got, want)
	}
}

func Test_ErrInvalidPayload(t *testing.T) {
	got := ErrInvalidPayload("Error for invalid nucleus payload")
	want := "Error for invalid nucleus payload"
	if got.Error() != want {
		t.Errorf("Received: %v, Expected: %v", got, want)
	}
}

func TestErrSecretNotFound(t *testing.T) {
	got := ErrSecretNotFound("SECRET_STRING")
	want := "secret with name SECRET_STRING not found"
	if got.Error() != want {
		t.Errorf("Received: %v, Expected: %v", got, want)
	}
}
