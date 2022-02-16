package secret

import (
	"errors"
	"log"
	"os"
	"testing"

	"github.com/LambdaTest/synapse/pkg/lumber"
)

func TestSecret_GetRepoSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}
	secretParser := New(logger)

	checkIncorrectPath := func(t *testing.T) {
		path := "../../testUtils/testdata/secretTestData/PathNotExist/a.json"
		secret, err := secretParser.GetRepoSecret(path)
		if secret != nil || err != nil {
			t.Errorf("Expected nil error and nil secret, recieved secret: %v, error: %v", secret, err)
		}
	}

	checkInvalidFile := func(t *testing.T) {
		path := "../../testUtils/testdata/secretTestData/invalidsecretfile"
		secret, err := secretParser.GetRepoSecret(path)

		if secret != nil {
			t.Errorf("Expected nil error and nil secret, recieved secret: %v, error: %v", secret, err)
		}
	}

	checkCorrectFile := func(t *testing.T) {
		path := "../../testUtils/testdata/secretTestData/secretfile.json"
		secret, err := secretParser.GetRepoSecret(path)

		if err != nil {
			t.Errorf("Recieved secret: %v, Received error: %v", secret, err)
		}
	}

	t.Run("TestSecret_GetRepoSecret for incorrect path", func(t *testing.T) {
		checkIncorrectPath(t)
	})
	t.Run("TestSecret_GetRepoSecret for invalid file", func(t *testing.T) {
		checkInvalidFile(t)
	})
	t.Run("TestSecret_GetRepoSecret for correct file", func(t *testing.T) {
		checkCorrectFile(t)
	})
}

func TestSecret_GetOauthSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}
	secretParser := New(logger)

	checkIncorrectPath := func(t *testing.T) {
		path := "../../testUtils/testdata/secretTestData/PathNotExist/a.json"
		oauth, err := secretParser.GetOauthSecret(path)
		if errors.Is(err, os.ErrNotExist) == false {
			t.Errorf("Expected nil error and nil secret, recieved secret: %v, error: %s", oauth, err)
		}
	}

	checkInvalidFile := func(t *testing.T) {
		path := "../../testUtils/testdata/secretTestData/invalidsecretfile"
		secret, err := secretParser.GetOauthSecret(path)

		if secret != nil {
			t.Errorf("Expected nil error and nil secret, recieved secret: %v, error: %v", secret, err)
		}
	}

	checkCorrectFile := func(t *testing.T) {
		path := "../../testUtils/testdata/secretTestData/secretfile.json"
		oauth, err := secretParser.GetOauthSecret(path)

		if err != nil {
			t.Errorf("Recieved secret: %v, Received error: %v", oauth, err)
		}
	}

	t.Run("TestSecret_GetOauthSecret for incorrect path", func(t *testing.T) {
		checkIncorrectPath(t)
	})
	t.Run("TestSecret_GetOauthSecret for invalid file", func(t *testing.T) {
		checkInvalidFile(t)
	})
	t.Run("TestSecret_GetOauthSecret for correct file", func(t *testing.T) {
		checkCorrectFile(t)
	})
}

func TestSubstituteSecret(t *testing.T) {
	logger, err := lumber.NewLogger(lumber.LoggingConfig{EnableConsole: true}, true, lumber.InstanceZapLogger)
	if err != nil {
		log.Fatalf("Could not instantiate logger %s", err.Error())
	}

	secretParser := New(logger)
	var expressions = []struct {
		params    map[string]string
		input     string
		output    string
		errorType error
	}{
		// basic
		{
			params:    map[string]string{"token": "secret"},
			input:     "${{ secrets.token }}",
			output:    "secret",
			errorType: nil,
		},
		// multiple
		{
			params:    map[string]string{"NPM_TOKEN": "secret", "TAG": "nucleus"},
			input:     "docker build --build-arg NPM_TOKEN=${{ secrets.NPM_TOKEN }} --tag=${{ secrets.TAG }}",
			output:    "docker build --build-arg NPM_TOKEN=secret --tag=nucleus",
			errorType: nil,
		},
		// no match
		{
			params:    map[string]string{"clone_token": "secret"},
			input:     "${{ secrets.token }}",
			output:    "${{ secrets.token }}",
			errorType: nil,
		},
	}

	for _, expr := range expressions {
		t.Run(expr.input, func(t *testing.T) {
			t.Logf(expr.input)
			output, err := secretParser.SubstituteSecret(expr.input, expr.params)
			if err != nil {
				if expr.errorType != nil {
					if err.Error() != expr.errorType.Error() {
						t.Errorf("Want error %q expanded but got error %q", expr.errorType, err)
						return
					}
					return
				}
				t.Errorf("Want %q expanded but got error %q", expr.input, err)
				return
			}

			if output != expr.output {
				t.Errorf("Want %q expanded to %q, got %q",
					expr.input,
					expr.output,
					output)
			}
		})
	}
}
