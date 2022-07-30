package secrets

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func removeCreatedPath(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
}

func TestGetLambdatestSecrets(t *testing.T) {
	lambdatestSecrets := secretsManager.GetLambdatestSecrets()
	assert.Equal(t, "dummysecretkey", lambdatestSecrets.SecretKey)
}
