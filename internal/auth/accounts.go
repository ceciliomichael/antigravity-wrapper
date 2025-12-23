package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Account represents a single account entry in accounts.json.
type Account struct {
	Email        string `json:"email"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Timestamp    int64  `json:"timestamp"`
	Expired      string `json:"expired"`
	ProjectID    string `json:"project_id,omitempty"`
}

// AccountsFile represents the structure of accounts.json.
type AccountsFile struct {
	Accounts     []Account `json:"accounts"`
	CurrentIndex int       `json:"current_index"`
}

// AccountManager handles round-robin account selection.
type AccountManager struct {
	mu           sync.Mutex
	filePath     string
	accounts     []Account
	currentIndex int
	tokenManager *TokenManager
}

// NewAccountManager creates a new AccountManager instance.
func NewAccountManager(filePath string, tokenManager *TokenManager) *AccountManager {
	return &AccountManager{
		filePath:     filePath,
		tokenManager: tokenManager,
	}
}

// Load reads accounts from the accounts.json file.
func (m *AccountManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("read accounts file: %w", err)
	}

	var accountsFile AccountsFile
	if err := json.Unmarshal(data, &accountsFile); err != nil {
		return fmt.Errorf("parse accounts file: %w", err)
	}

	if len(accountsFile.Accounts) == 0 {
		return fmt.Errorf("no accounts found in %s", m.filePath)
	}

	m.accounts = accountsFile.Accounts
	m.currentIndex = accountsFile.CurrentIndex

	// Ensure current_index is within bounds
	if m.currentIndex < 0 || m.currentIndex >= len(m.accounts) {
		m.currentIndex = 0
	}

	log.Infof("Loaded %d accounts from %s (current index: %d)", len(m.accounts), m.filePath, m.currentIndex)
	return nil
}

// Next returns the next account in round-robin order and advances the index.
func (m *AccountManager) Next() (*Credentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.accounts) == 0 {
		return nil, fmt.Errorf("no accounts available")
	}

	// Get current account
	account := m.accounts[m.currentIndex]
	creds := m.toCredentials(&account)

	// Log which account is being used
	log.Infof("Using account: %s (index: %d/%d)", account.Email, m.currentIndex, len(m.accounts))

	// Advance index for next request (round-robin)
	m.currentIndex = (m.currentIndex + 1) % len(m.accounts)

	return creds, nil
}

// toCredentials converts an Account to Credentials.
func (m *AccountManager) toCredentials(account *Account) *Credentials {
	return &Credentials{
		Type:         "antigravity",
		AccessToken:  account.AccessToken,
		RefreshToken: account.RefreshToken,
		ExpiresIn:    account.ExpiresIn,
		Timestamp:    account.Timestamp,
		Expired:      account.Expired,
		Email:        account.Email,
		ProjectID:    account.ProjectID,
	}
}

// SaveState persists the current state (index) back to the accounts.json file.
func (m *AccountManager) SaveState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read current file to preserve all data
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("read accounts file: %w", err)
	}

	var accountsFile AccountsFile
	if err := json.Unmarshal(data, &accountsFile); err != nil {
		return fmt.Errorf("parse accounts file: %w", err)
	}

	// Update only the current_index
	accountsFile.CurrentIndex = m.currentIndex

	// Write back
	updatedData, err := json.MarshalIndent(accountsFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal accounts file: %w", err)
	}

	if err := os.WriteFile(m.filePath, updatedData, 0600); err != nil {
		return fmt.Errorf("write accounts file: %w", err)
	}

	return nil
}

// Count returns the number of loaded accounts.
func (m *AccountManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.accounts)
}

// CurrentEmail returns the email of the current account (for logging).
func (m *AccountManager) CurrentEmail() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.accounts) == 0 {
		return ""
	}
	idx := m.currentIndex
	if idx < 0 || idx >= len(m.accounts) {
		idx = 0
	}
	return m.accounts[idx].Email
}

// DefaultAccountsPath returns the default path for accounts.json.
func DefaultAccountsPath() string {
	// Check current directory first
	localPath := filepath.Join(".antigravity-wrapper", "accounts.json")
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// Fall back to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return localPath
	}
	return filepath.Join(home, ".antigravity-wrapper", "accounts.json")
}

// LoadAccountManager attempts to load an AccountManager from the default path.
// Returns nil if no accounts.json file exists.
func LoadAccountManager(tokenManager *TokenManager) *AccountManager {
	path := DefaultAccountsPath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Debugf("No accounts.json found at %s", path)
		return nil
	}

	manager := NewAccountManager(path, tokenManager)
	if err := manager.Load(); err != nil {
		log.Warnf("Failed to load accounts: %v", err)
		return nil
	}

	return manager
}
