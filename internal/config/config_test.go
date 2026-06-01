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

func TestWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	expected := Config{
		Credentials: map[string]Credential{
			"test@test.com": {
				Key:     "api-key-123",
				Token:   "token",
				Credits: 100,
				Status:  true,
			},
		},
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

func TestRead_FileNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	expected := Config{Credentials: map[string]Credential{}}

	cfg, err := read(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(expected, cfg) {
		t.Errorf("Returned config does not match expected. Got: %+v, Want: %+v", cfg, expected)
	}
}

func TestRead_FileExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	expected := Config{
		Credentials: map[string]Credential{
			"test@test.com": {
				Key:     "api-key-123",
				Token:   "token",
				Credits: 100,
				Status:  true,
			},
		},
	}

	write(path, expected)
	cfg, err := read(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(expected, cfg) {
		t.Errorf("Returned config does not match expected. Got: %+v, Want: %+v", cfg, expected)
	}
}

func TestAddCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	cfg := Config{Credentials: map[string]Credential{}}

	cred := Credential{Key: "api-key-123", Token: "abc-123", Credits: 100}

	err := cfg.addCredential(path, "test@test.com", cred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cred.Status = true
	expected := Config{Credentials: map[string]Credential{"test@test.com": cred}}

	if !reflect.DeepEqual(expected, cfg) {
		t.Errorf("Config mismatch.\nGot:  %+v\nWant: %+v", cfg, expected)
	}
}

func TestAddCredential_ExclusiveStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	cfg := Config{
		Credentials: map[string]Credential{
			"existing@test.com": {
				Key:     "api-key-123",
				Token:   "token",
				Credits: 100,
				Status:  true,
			},
		},
	}

	newCred := Credential{Key: "api-key-456", Token: "abc-456", Credits: 200}
	err := cfg.addCredential(path, "new@test.com", newCred)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Credentials["new@test.com"].Status {
		t.Error("Expected new credential to have Status: false")
	}

	if !cfg.Credentials["existing@test.com"].Status {
		t.Error("Expected previous active credential to be set to Status: true")
	}
}

func TestDeleteCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	cfg := Config{
		Credentials: map[string]Credential{
			"credential1": {},
		},
	}
	expected := Config{Credentials: map[string]Credential{}}

	err := cfg.deleteCredential(path, "credential1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(expected, cfg) {
		t.Errorf("Config mismatch.\nGot:  %+v\nWant: %+v", cfg, expected)
	}
}

func TestDeleteCredential_ExclusiveStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	cfg := Config{
		Credentials: map[string]Credential{
			"credential1": {Status: true},
			"credential2": {Status: false},
		},
	}
	expected := Config{Credentials: map[string]Credential{"credential2": {Status: true}}}

	err := cfg.deleteCredential(path, "credential1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(expected, cfg) {
		t.Errorf("Config mismatch.\nGot:  %+v\nWant: %+v", cfg, expected)
	}
}

func TestGetCredentials(t *testing.T) {
	cfg := Config{
		Credentials: map[string]Credential{
			"credential1": {},
			"credential2": {Status: true},
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

func TestActivateCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), configFileName)
	cfg := Config{
		Credentials: map[string]Credential{
			"credential1": {Status: true},
			"credential2": {Status: false},
		},
	}
	expected := Config{Credentials: map[string]Credential{
		"credential1": {Status: false},
		"credential2": {Status: true},
	}}

	err := cfg.activateCredential(path, "credential2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(expected, cfg) {
		t.Errorf("Config mismatch.\nGot:  %+v\nWant: %+v", cfg, expected)
	}
}

func TestGetCredential(t *testing.T) {
	cfg := Config{
		Credentials: map[string]Credential{
			"credential1": {Token: "abc-123", Status: false},
			"credential2": {Token: "abc-456", Status: true},
		},
	}
	expectedKey := "credential2"
	expected := Credential{Token: "abc-456", Status: true}

	key, credential, err := cfg.GetCredential()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if key != expectedKey {
		t.Errorf("Credential key mismatch.\nGot:  %+v\nWant: %+v", key, expectedKey)
	}

	if !reflect.DeepEqual(expected, credential) {
		t.Errorf("Credential mismatch.\nGot:  %+v\nWant: %+v", credential, expected)
	}
}

func TestGetCredential_WithoutStatus(t *testing.T) {
	cfg := Config{
		Credentials: map[string]Credential{
			"credential1": {Token: "abc-123"},
		},
	}

	_, _, err := cfg.GetCredential()
	if err == nil {
		t.Errorf("Expected an error for negative input, but got nil")
	}
}
