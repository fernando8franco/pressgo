package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	configDir      = "pressgo"
	configFileName = ".config.json"
	activeEmoji    = "✅"
	inactiveEmoji  = "❌"
)

type Config struct {
	Credentials map[string]Credential `json:"credentials"`
}

type Credential struct {
	Key     string `json:"key"`
	Token   string `json:"token"`
	Credits int    `json:"credits"`
	Status  bool   `json:"status"`
}

func CreateCredential(key, token string, credits int) Credential {
	return Credential{
		Key:     key,
		Token:   token,
		Credits: credits,
		Status:  false,
	}
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configFilePath := filepath.Join(homeDir, configDir, configFileName)
	return configFilePath, nil
}

func write(configFilePath string, cfg Config) error {
	dir := filepath.Dir(configFilePath)
	if _, err := os.Stat(dir); err != nil && errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
	}

	configFile, err := os.Create(configFilePath)
	if err != nil {
		return err
	}
	defer configFile.Close()

	encoder := json.NewEncoder(configFile)
	encoder.SetIndent("", "\t")

	if err := encoder.Encode(cfg); err != nil {
		return err
	}

	return nil
}

func read(configFilePath string) (Config, error) {
	if _, err := os.Stat(configFilePath); err != nil && errors.Is(err, os.ErrNotExist) {
		write(configFilePath, Config{Credentials: map[string]Credential{}})
	}

	jsonFile, err := os.Open(configFilePath)
	if err != nil {
		return Config{}, err
	}
	defer jsonFile.Close()

	var cfg Config
	if err := json.NewDecoder(jsonFile).Decode(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Read() (Config, error) {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	cfg, err := read(configFilePath)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c *Config) addCredential(configFilePath, id string, credential Credential) error {
	if len(c.Credentials) == 0 {
		credential.Status = true
	}

	c.Credentials[id] = credential
	return write(configFilePath, *c)
}

func (c *Config) AddCredential(id string, credential Credential) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	return c.addCredential(configFilePath, id, credential)
}

func (c *Config) deleteCredential(configFilePath, id string) error {
	if _, ok := c.Credentials[id]; !ok {
		return fmt.Errorf("The credential id doesn't exist")
	}
	delete(c.Credentials, id)

	for key, value := range c.Credentials {
		value.Status = true
		c.Credentials[key] = value
		break
	}
	return write(configFilePath, *c)
}

func (c *Config) DeleteCredential(id string) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	return c.deleteCredential(configFilePath, id)
}

func (c *Config) activateCredential(configFilePath, id string) error {
	if _, ok := c.Credentials[id]; !ok {
		return fmt.Errorf("The credential id doesn't exist")
	}

	for key, value := range c.Credentials {
		value.Status = false
		if key == id {
			value.Status = true
		}
		c.Credentials[key] = value
	}

	return write(configFilePath, *c)
}

func (c *Config) ActivateCredential(id string) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	return c.activateCredential(configFilePath, id)
}

type CredentialWithID struct {
	ID string
	Credential
}

func (c *Config) GetCredentials() [][]string {
	var credentials [][]string
	for key, value := range c.Credentials {
		status := inactiveEmoji
		if value.Status {
			status = activeEmoji
		}
		row := []string{
			key,
			fmt.Sprintf("%s...", safeTruncate(value.Key, 20)),
			strconv.Itoa(value.Credits),
			status,
		}
		credentials = append(credentials, row)
	}

	slices.SortFunc(credentials, func(a, b []string) int {
		if a[3] != b[3] {
			if a[3] == activeEmoji {
				return -1
			}
			return 1
		}
		creditsA, _ := strconv.Atoi(a[2])
		creditsB, _ := strconv.Atoi(b[2])
		if creditsA != creditsB {
			return creditsB - creditsA
		}
		return strings.Compare(a[0], b[0])
	})

	return credentials
}

func safeTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (c *Config) GetCredential() (string, Credential, error) {
	var foundKey string

	if len(c.Credentials) == 0 {
		return "", Credential{}, fmt.Errorf("no credentials found")
	}

	for key, value := range c.Credentials {
		if value.Status {
			foundKey = key
			break
		}
	}

	if foundKey == "" {
		return "", Credential{}, fmt.Errorf("no active credential found")
	}

	return foundKey, c.Credentials[foundKey], nil
}
