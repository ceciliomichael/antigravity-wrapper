package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	apiKeysFilename = "api_keys.json"
)

// APIKey represents a generated API key.
type APIKey struct {
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	Note      string    `json:"note,omitempty"`
}

// KeyStore manages API key persistence and validation.
type KeyStore struct {
	dir  string
	path string
	keys map[string]*APIKey
	mu   sync.RWMutex
}

// NewKeyStore creates a new API key store.
func NewKeyStore(dir string) (*KeyStore, error) {
	if dir == "" {
		return nil, fmt.Errorf("data directory cannot be empty")
	}

	ks := &KeyStore{
		dir:  dir,
		path: filepath.Join(dir, apiKeysFilename),
		keys: make(map[string]*APIKey),
	}

	if err := ks.load(); err != nil {
		return nil, err
	}

	return ks, nil
}

// Generate creates a new API key and saves it to the store.
func (ks *KeyStore) Generate(note string) (*APIKey, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	key := uuid.New().String()
	apiKey := &APIKey{
		Key:       key,
		CreatedAt: time.Now(),
		Note:      note,
	}

	ks.keys[key] = apiKey

	if err := ks.save(); err != nil {
		delete(ks.keys, key) // Rollback on failure
		return nil, fmt.Errorf("save keys: %w", err)
	}

	return apiKey, nil
}

// Validate checks if the provided API key is valid.
func (ks *KeyStore) Validate(key string) bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	_, exists := ks.keys[key]
	return exists
}

// List returns all stored API keys.
func (ks *KeyStore) List() []*APIKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	list := make([]*APIKey, 0, len(ks.keys))
	for _, k := range ks.keys {
		list = append(list, k)
	}
	return list
}

// Revoke removes an API key from the store.
func (ks *KeyStore) Revoke(key string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if _, exists := ks.keys[key]; !exists {
		return fmt.Errorf("key not found")
	}

	original := ks.keys[key]
	delete(ks.keys, key)

	if err := ks.save(); err != nil {
		ks.keys[key] = original // Rollback
		return fmt.Errorf("save keys: %w", err)
	}

	return nil
}

// load reads keys from the JSON file.
func (ks *KeyStore) load() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	data, err := os.ReadFile(ks.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No keys yet
		}
		return fmt.Errorf("read keys file: %w", err)
	}

	var storedKeys []*APIKey
	if err := json.Unmarshal(data, &storedKeys); err != nil {
		return fmt.Errorf("parse keys file: %w", err)
	}

	for _, k := range storedKeys {
		ks.keys[k.Key] = k
	}

	return nil
}

// save writes keys to the JSON file.
func (ks *KeyStore) save() error {
	// Note: Caller must hold lock

	list := make([]*APIKey, 0, len(ks.keys))
	for _, k := range ks.keys {
		list = append(list, k)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal keys: %w", err)
	}

	if err := os.WriteFile(ks.path, data, 0600); err != nil {
		return fmt.Errorf("write keys file: %w", err)
	}

	return nil
}
