package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGetConfigFilePath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, configDir, configFileName)

	got, err := getConfigFilePath()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestSortCredentials(t *testing.T) {
	tests := []struct {
		name     string
		input    []Credential
		expected []Credential
	}{
		{
			name: "active status sorts before inactive",
			input: []Credential{
				{ID: "Alice", Credits: 100, Status: false},
				{ID: "Bob", Credits: 100, Status: true},
			},
			expected: []Credential{
				{ID: "Bob", Credits: 100, Status: true},
				{ID: "Alice", Credits: 100, Status: false},
			},
		},
		{
			name: "higher credits sorts first when status is equal",
			input: []Credential{
				{ID: "Alice", Credits: 100, Status: false},
				{ID: "Bob", Credits: 200, Status: false},
			},
			expected: []Credential{
				{ID: "Bob", Credits: 200, Status: false},
				{ID: "Alice", Credits: 100, Status: false},
			},
		},
		{
			name: "alphabetical order when status and credits are equal",
			input: []Credential{
				{ID: "Charlie", Credits: 100, Status: false},
				{ID: "Alice", Credits: 100, Status: false},
				{ID: "Bob", Credits: 100, Status: false},
			},
			expected: []Credential{
				{ID: "Alice", Credits: 100, Status: false},
				{ID: "Bob", Credits: 100, Status: false},
				{ID: "Charlie", Credits: 100, Status: false},
			},
		},
		{
			name: "all criteria combined",
			input: []Credential{
				{ID: "Charlie", Credits: 100, Status: false},
				{ID: "Alice", Credits: 200, Status: false},
				{ID: "Bob", Credits: 100, Status: false},
				{ID: "Zach", Credits: 200, Status: true},
			},
			expected: []Credential{
				{ID: "Zach", Credits: 200, Status: true},
				{ID: "Alice", Credits: 200, Status: false},
				{ID: "Bob", Credits: 100, Status: false},
				{ID: "Charlie", Credits: 100, Status: false},
			},
		},
		{
			name:     "empty slice",
			input:    []Credential{},
			expected: []Credential{},
		},
		{
			name: "single element",
			input: []Credential{
				{ID: "Alice", Credits: 100, Status: false},
			},
			expected: []Credential{
				{ID: "Alice", Credits: 100, Status: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]Credential, len(tt.input))
			copy(input, tt.input)

			sortCredentials(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("\ngot  %+v\nwant %+v", input, tt.expected)
			}
		})
	}
}

func TestVerifyStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    []Credential
		expected []Credential
	}{
		{
			name:     "empty slice",
			input:    []Credential{},
			expected: []Credential{},
		},
		{
			name: "single element",
			input: []Credential{
				{Status: true},
			},
			expected: []Credential{
				{Status: true},
			},
		},
		{
			name: "multiple elements with one true",
			input: []Credential{
				{Status: true}, {Status: false}, {Status: false},
			},
			expected: []Credential{
				{Status: true}, {Status: false}, {Status: false},
			},
		},
		{
			name: "multiple elements with two trues",
			input: []Credential{
				{Status: true}, {Status: true}, {Status: false},
			},
			expected: []Credential{
				{Status: true}, {Status: false}, {Status: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]Credential, len(tt.input))
			copy(input, tt.input)

			verifyStatus(input)

			if !reflect.DeepEqual(input, tt.expected) {
				t.Errorf("\ngot  %+v\nwant %+v", input, tt.expected)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	expected := Config{
		Credentials: []Credential{{ID: "test1", Key: "api-key-123", Token: "token", Credits: 100, Status: true}},
	}

	err := write(path, expected)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file was not created: %v", err)
	}

	var cfg Config
	json.Unmarshal(data, &cfg)
	if !reflect.DeepEqual(expected, cfg) {
		t.Errorf("Returned config does not match expected. Got: %+v, Want: %+v", cfg, expected)
	}
}

func TestRead(t *testing.T) {
	existingConfig := Config{
		Credentials: []Credential{
			{ID: "test1", Key: "api-key-123", Token: "token", Credits: 100, Status: true},
		},
	}

	tests := []struct {
		name     string
		setup    func(path string)
		expected Config
	}{
		{
			name:     "file doesn't exist",
			setup:    func(path string) {},
			expected: Config{Credentials: []Credential{}},
		},
		{
			name:     "file exists",
			setup:    func(path string) { write(path, existingConfig) },
			expected: existingConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), configFileName)
			tt.setup(path)

			cfg, err := read(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.expected, cfg) {
				t.Errorf("\ngot %+v,\nwant %+v", cfg, tt.expected)
			}
		})
	}
}

func TestAddCredential(t *testing.T) {
	credential1 := Credential{ID: "test1", Key: "api-key-123", Token: "token", Credits: 100, Status: true}
	credential2 := Credential{ID: "test2", Key: "api-key-456", Token: "token", Credits: 200}
	credential2True := Credential{ID: "test2", Key: "api-key-456", Token: "token", Credits: 200, Status: true}

	tests := []struct {
		name          string
		config        Config
		addCredential Credential
		expected      []Credential
	}{
		{
			name:          "empty config credentials",
			config:        Config{Credentials: []Credential{}},
			addCredential: credential1,
			expected:      []Credential{credential1},
		},
		{
			name:          "check exclusive status with status credential false",
			config:        Config{Credentials: []Credential{credential1}},
			addCredential: credential2,
			expected:      []Credential{credential1, credential2},
		},
		{
			name:          "check exclusive status with status credential true",
			config:        Config{Credentials: []Credential{credential1}},
			addCredential: credential2True,
			expected:      []Credential{credential1, credential2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), configFileName)

			err := tt.config.addCredential(path, tt.addCredential)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.config.Credentials, tt.expected) {
				t.Errorf("\ngot %+v,\nwant %+v", tt.config.Credentials, tt.expected)
			}
		})
	}
}

func TestDeleteCredential(t *testing.T) {
	credential1 := Credential{ID: "test1", Key: "api-key-123", Token: "token", Credits: 100, Status: true}
	credential2 := Credential{ID: "test2", Key: "api-key-456", Token: "token", Credits: 200, Status: false}
	promotedInactiveCred := Credential{ID: "test2", Key: "api-key-456", Token: "token", Credits: 200, Status: true}

	tests := []struct {
		name             string
		config           Config
		deleteCredential string
		expectedErr      bool
		expected         []Credential
	}{
		{
			name:             "empty config credentials returns error",
			config:           Config{Credentials: []Credential{}},
			deleteCredential: credential1.ID,
			expectedErr:      true,
			expected:         []Credential{},
		},
		{
			name:             "delete the only credential leaves config empty",
			config:           Config{Credentials: []Credential{credential1}},
			deleteCredential: credential1.ID,
			expectedErr:      false,
			expected:         []Credential{},
		},
		{
			name:             "delete active credential promotes another to active",
			config:           Config{Credentials: []Credential{credential1, credential2}},
			deleteCredential: credential1.ID,
			expectedErr:      false,
			expected:         []Credential{promotedInactiveCred},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), configFileName)

			err := tt.config.deleteCredential(path, tt.deleteCredential)
			if tt.expectedErr {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.config.Credentials, tt.expected) {
				t.Errorf("\ngot %+v,\nwant %+v", tt.config.Credentials, tt.expected)
			}
		})
	}
}

func TestActivateCredential(t *testing.T) {
	credential1 := Credential{ID: "test1", Key: "api-key-123", Token: "token", Credits: 100, Status: true}
	credential2 := Credential{ID: "test2", Key: "api-key-456", Token: "token", Credits: 200, Status: false}

	inactiveCred1 := Credential{ID: "test1", Key: "api-key-123", Token: "token", Credits: 100, Status: false}
	activeCred2 := Credential{ID: "test2", Key: "api-key-456", Token: "token", Credits: 200, Status: true}

	tests := []struct {
		name             string
		config           Config
		activeCredential string
		expectedErr      bool
		expected         []Credential
	}{
		{
			name:             "empty config credentials returns error",
			config:           Config{Credentials: []Credential{}},
			activeCredential: credential1.ID,
			expectedErr:      true,
			expected:         []Credential{},
		},
		{
			name:             "activate credential",
			config:           Config{Credentials: []Credential{credential1, credential2}},
			activeCredential: credential2.ID,
			expectedErr:      false,
			expected:         []Credential{activeCred2, inactiveCred1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), configFileName)

			err := tt.config.activateCredential(path, tt.activeCredential)
			if tt.expectedErr {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tt.config.Credentials, tt.expected) {
				t.Errorf("\ngot %+v,\nwant %+v", tt.config.Credentials, tt.expected)
			}
		})
	}
}

func TestGetCredentials(t *testing.T) {
	cfg := Config{
		Credentials: []Credential{
			{ID: "credential2", Status: true},
			{ID: "credential1"},
		},
	}
	expected := [][]string{
		{"credential2", "...", "0", "✅"},
		{"credential1", "...", "0", "❌"},
	}

	got := cfg.GetCredentials()

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Config mismatch.\nGot:  %+v\nWant: %+v", got, expected)
	}
}

func TestGetCredential(t *testing.T) {
	cfg := Config{
		Credentials: []Credential{
			{ID: "credential1", Status: true},
			{ID: "credential2", Status: false},
		},
	}

	expected := Credential{ID: "credential1", Status: true}

	got, err := cfg.GetCredential()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("Config mismatch.\nGot:  %+v\nWant: %+v", got, expected)
	}
}
