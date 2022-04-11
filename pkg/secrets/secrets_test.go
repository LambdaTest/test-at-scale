package secrets

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/LambdaTest/test-at-scale/pkg/global"
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

func TestWriteGitSecrets(t *testing.T) {
	expectedFile := fmt.Sprintf("%s/%s", testdDataDir, global.GitConfigFileName)
	defer removeCreatedPath(testdDataDir)
	expectedFileContent := `{"access_token":"dummytoken","expiry":"0001-01-01T00:00:00Z","refresh_token":"","token_type":"Bearer"}`
	err := secretsManager.WriteGitSecrets(testdDataDir)
	if err != nil {
		t.Errorf("error while writing secrets: %v", err)
	}
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("could not find the git config file: %v", err)
	}

	fileContent, err := ioutil.ReadFile(expectedFile)
	if err != nil {
		t.Errorf("error reading git config file: %v", err)
	}
	assert.Equal(t, expectedFileContent, string(fileContent))
}
