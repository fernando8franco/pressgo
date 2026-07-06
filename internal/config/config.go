package config

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
)

const (
	configDir      = "pressgo"
	configFileName = ".config.json"
	activeEmoji    = "✅"
	inactiveEmoji  = "❌"
)

type Config struct {
	Credentials []Credential `json:"credentials"`
}

type Credential struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Token   string `json:"token"`
	Credits int    `json:"credits"`
	Status  bool   `json:"status"`
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configFilePath := filepath.Join(homeDir, configDir, configFileName)
	return configFilePath, nil
}

func sortCredentials(credentials []Credential) {
	slices.SortFunc(credentials, func(a, b Credential) int {
		if a.Status != b.Status {
			if a.Status {
				return -1
			}
			return 1
		}

		if a.Credits != b.Credits {
			return cmp.Compare(b.Credits, a.Credits)
		}

		return cmp.Compare(a.ID, b.ID)
	})
}

func verifyStatus(credentials []Credential) {
	for i := 0; i < len(credentials); i++ {
		credentials[i].Status = false
		if i == 0 {
			credentials[0].Status = true
		}
	}
}

func write(configFilePath string, cfg Config) error {
	verifyStatus(cfg.Credentials)
	sortCredentials(cfg.Credentials)

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
		write(configFilePath, Config{Credentials: []Credential{}})
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

func (c *Config) addCredential(configFilePath string, credential Credential) error {
	credential.Status = false
	c.Credentials = append(c.Credentials, credential)
	return write(configFilePath, *c)
}

func (c *Config) AddCredential(credential Credential) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	return c.addCredential(configFilePath, credential)
}

func (c *Config) deleteCredential(configFilePath, id string) error {
	index := slices.IndexFunc(c.Credentials, func(crd Credential) bool {
		return crd.ID == id
	})
	if index == -1 {
		return fmt.Errorf("The credential id doesn't exist")
	}

	c.Credentials = slices.Delete(c.Credentials, index, index+1)

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
	index := slices.IndexFunc(c.Credentials, func(crd Credential) bool {
		return crd.ID == id
	})
	if index == -1 {
		return fmt.Errorf("The credential id doesn't exist")
	}

	c.Credentials[index].Status = true

	aux := c.Credentials[index]
	c.Credentials[index] = c.Credentials[0]
	c.Credentials[0] = aux

	return write(configFilePath, *c)
}

func (c *Config) ActivateCredential(id string) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	return c.activateCredential(configFilePath, id)
}

func (c *Config) GetCredentials() [][]string {
	var credentials [][]string
	for _, value := range c.Credentials {
		status := inactiveEmoji
		if value.Status {
			status = activeEmoji
		}
		row := []string{
			value.ID,
			fmt.Sprintf("%s...", safeTruncate(value.Key, 20)),
			strconv.Itoa(value.Credits),
			status,
		}
		credentials = append(credentials, row)
	}

	return credentials
}

func safeTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (c *Config) GetCredential() (Credential, error) {
	if len(c.Credentials) == 0 {
		return Credential{}, fmt.Errorf("no credentials found")
	}

	return c.Credentials[0], nil
}

func (c *Config) updateTokenCredential(configFilePath, token string) error {
	if len(c.Credentials) == 0 {
		return fmt.Errorf("no credentials found")
	}
	if token == "" {
		return fmt.Errorf("The token can't be empty")
	}

	c.Credentials[0].Token = token

	return write(configFilePath, *c)
}

func (c *Config) UpdateTokenCredential(token string) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	return c.updateTokenCredential(configFilePath, token)
}
