package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudivision/cloudivision/internal/redact"
)

const (
	DockerConfigJSONKey = ".dockerconfigjson"
	DockerConfigKey     = "config.json"
	UsernameKey         = "username"
	PasswordKey         = "password"
	TokenKey            = "token"
)

func ParseCredential(data map[string][]byte) (Credential, error) {
	credential := Credential{
		Username: strings.TrimSpace(string(data[UsernameKey])),
		Password: strings.TrimSpace(string(data[PasswordKey])),
		Token:    strings.TrimSpace(string(data[TokenKey])),
	}
	if config := data[DockerConfigJSONKey]; len(config) > 0 {
		credential.DockerConfigJSON = append([]byte(nil), config...)
	} else if config := data[DockerConfigKey]; len(config) > 0 {
		credential.DockerConfigJSON = append([]byte(nil), config...)
	}
	if credential.Empty() {
		return Credential{}, ErrCredentialsMissing
	}
	if len(credential.DockerConfigJSON) > 0 {
		if _, err := dockerConfigJSON("validation.invalid", credential); err != nil {
			return Credential{}, err
		}
		return credential, nil
	}
	if credential.Password != "" && credential.Username == "" {
		return Credential{}, fmt.Errorf("username is required with password: %w", ErrCredentialsInvalid)
	}
	if credential.Password == "" && credential.Token == "" {
		return Credential{}, fmt.Errorf("password or token is required: %w", ErrCredentialsInvalid)
	}
	return credential, nil
}

func LoadCredentialDir(dir string) (Credential, error) {
	data := map[string][]byte{}
	for _, key := range []string{DockerConfigJSONKey, DockerConfigKey, UsernameKey, PasswordKey, TokenKey} {
		value, err := os.ReadFile(filepath.Join(dir, key))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Credential{}, fmt.Errorf("read registry credential key %q: %w", key, err)
		}
		data[key] = value
	}
	return ParseCredential(data)
}

func WriteDockerConfig(dir string, result *LoginResult) (string, error) {
	if result == nil || len(result.DockerConfigJSON) == 0 {
		return "", fmt.Errorf("login returned no docker config: %w", ErrCredentialsInvalid)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create Docker config directory: %w", err)
	}
	path := filepath.Join(dir, DockerConfigKey)
	if err := os.WriteFile(path, result.DockerConfigJSON, 0o600); err != nil {
		return "", fmt.Errorf("write Docker config: %w", err)
	}
	return path, nil
}

func Redactor(credential Credential) redact.Redactor {
	values := []string{credential.Username, credential.Password, credential.Token}
	secret := credential.Password
	if credential.Token != "" {
		secret = credential.Token
	}
	if secret != "" {
		username := credential.Username
		if username == "" {
			username = "oauth2"
		}
		values = append(values,
			username+":"+secret,
			base64.StdEncoding.EncodeToString([]byte(username+":"+secret)),
		)
	}
	if len(credential.DockerConfigJSON) > 0 {
		values = append(values, string(credential.DockerConfigJSON))
		var config struct {
			Auths map[string]struct {
				Auth     string `json:"auth"`
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"auths"`
		}
		if json.Unmarshal(credential.DockerConfigJSON, &config) == nil {
			for _, auth := range config.Auths {
				values = append(values, auth.Auth, auth.Username, auth.Password)
				if decoded, err := base64.StdEncoding.DecodeString(auth.Auth); err == nil {
					values = append(values, string(decoded))
				}
			}
		}
	}
	return redact.New(values...)
}
