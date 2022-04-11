package secret

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/LambdaTest/test-at-scale/pkg/core"
	"github.com/LambdaTest/test-at-scale/pkg/errs"
	"github.com/LambdaTest/test-at-scale/pkg/global"
	"github.com/LambdaTest/test-at-scale/pkg/lumber"
)

type secretParser struct {
	logger      lumber.Logger
	secretRegex *regexp.Regexp
}

// New return new secret parser
func New(logger lumber.Logger) core.SecretParser {
	return &secretParser{
		logger:      logger,
		secretRegex: regexp.MustCompile(global.SecretRegex),
	}
}

// GetRepoSecret read repo secrets from given path
func (s *secretParser) GetRepoSecret(path string) (map[string]string, error) {
	var secretData map[string]string
	if _, err := os.Stat(path); os.IsNotExist(err) {
		s.logger.Debugf("failed to find user env secrets in path %s, as path does not exists", path)
		return nil, nil
	}
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, &secretData); err != nil {
		s.logger.Errorf("failed to unmarshal user env secrets, error %v", err)
		return nil, errs.ErrUnMarshalJSON
	}

	// extract secretmap from data map[data: map[secretname:secretvalue]]
	return secretData, nil
}

// GetOauthSecret parses the oauth secret
func (s *secretParser) GetOauthSecret(path string) (*core.Oauth, error) {
	o := &core.Oauth{
		Type: core.Bearer,
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		s.logger.Errorf("failed to find oauth secret in path %s", path)
		return nil, err
	}
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(body, o); err != nil {
		s.logger.Errorf("failed to unmarshal oauth secret, error %v", err)
		return nil, errs.ErrUnMarshalJSON
	}
	if o.AccessToken == "" {
		return nil, errs.ErrMissingAccessToken
	}
	// If tokentype is not basic set it to bearer
	if o.Type != core.Basic {
		o.Type = core.Bearer
	}

	return o, err
}

// SubstituteSecret replace secret placeholders with their respective values
func (s *secretParser) SubstituteSecret(command string, secretData map[string]string) (string, error) {
	matches := s.secretRegex.FindAllStringSubmatch(command, -1)
	if matches == nil {
		return command, nil
	}
	result := command
	for _, match := range matches {
		if len(match) < 2 {
			return "", errs.ErrSecretRegexMatch
		}
		// validating secret key exists or not
		if _, ok := secretData[match[1]]; !ok {
			s.logger.Warnf("secret with name %s not found in map", match[0])
			continue
		}
		result = strings.ReplaceAll(result, match[0], secretData[match[1]])
	}

	return result, nil
}

func (s *secretParser) Expired(token *core.Oauth) bool {
	if token.RefreshToken == "" {
		return false
	}
	if token.Expiry.IsZero() && token.AccessToken != "" {
		return false
	}
	return token.Expiry.Add(-global.ExpiryDelta).
		Before(time.Now())
}
