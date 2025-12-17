package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store handles credential persistence to the filesystem.
type Store struct {
	dir string
}

// NewStore creates a new credential store at the specified directory.
func NewStore(dir string) *Store {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			dir = ".antigravity"
		} else {
			dir = filepath.Join(home, ".antigravity")
		}
	}
	return &Store{dir: dir}
}

// EnsureDir creates the credentials directory if it doesn't exist.
func (s *Store) EnsureDir() error {
	return os.MkdirAll(s.dir, 0700)
}

// Save persists credentials to a file.
func (s *Store) Save(creds *Credentials) (string, error) {
	if err := s.EnsureDir(); err != nil {
		return "", fmt.Errorf("create credentials directory: %w", err)
	}

	filename := s.filenameForCredentials(creds)
	path := filepath.Join(s.dir, filename)

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("write credentials file: %w", err)
	}

	return path, nil
}

// Load reads credentials from a file.
func (s *Store) Load(filename string) (*Credentials, error) {
	path := filepath.Join(s.dir, filename)
	return s.LoadPath(path)
}

// LoadPath reads credentials from a full path.
func (s *Store) LoadPath(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials file: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	return &creds, nil
}

// LoadFirst attempts to load the first available credentials file.
func (s *Store) LoadFirst() (*Credentials, string, error) {
	files, err := s.List()
	if err != nil {
		return nil, "", err
	}

	if len(files) == 0 {
		return nil, "", fmt.Errorf("no credentials found in %s", s.dir)
	}

	creds, err := s.Load(files[0])
	if err != nil {
		return nil, "", err
	}

	return creds, files[0], nil
}

// List returns all credential files in the store.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read credentials directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "antigravity") && strings.HasSuffix(name, ".json") {
			files = append(files, name)
		}
	}

	return files, nil
}

// Delete removes a credentials file.
func (s *Store) Delete(filename string) error {
	path := filepath.Join(s.dir, filename)
	return os.Remove(path)
}

// filenameForCredentials generates a filename based on the email.
func (s *Store) filenameForCredentials(creds *Credentials) string {
	if creds.Email == "" {
		return "antigravity.json"
	}
	sanitized := strings.ReplaceAll(creds.Email, "@", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	return fmt.Sprintf("antigravity-%s.json", sanitized)
}

// Update saves updated credentials back to the store.
func (s *Store) Update(creds *Credentials) error {
	_, err := s.Save(creds)
	return err
}
