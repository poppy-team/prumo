package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrNoAvailableAccount = errors.New("gateway: no healthy account available in account pool")
	ErrAccountNotFound    = errors.New("gateway: account not found")
)

// Account represents one credentialed account for a provider.
type Account struct {
	ID            string    `json:"id"`
	Provider      string    `json:"provider"`
	KeyHash       string    `json:"key_hash"`
	Status        string    `json:"status"` // healthy, cooldown, exhausted
	CooldownUntil time.Time `json:"cooldown_until,omitempty"`
	Failures      int       `json:"failures"`
	TotalRequests int       `json:"total_requests"`
}

// AccountPool manages multiple accounts across providers.
type AccountPool struct {
	mu       sync.RWMutex
	accounts map[string]*Account          // accountID -> Account
	byProv   map[string][]*Account        // provider -> list of accounts
	indices  map[string]int               // provider -> round-robin cursor
}

// NewAccountPool creates an empty account pool.
func NewAccountPool() *AccountPool {
	return &AccountPool{
		accounts: make(map[string]*Account),
		byProv:   make(map[string][]*Account),
		indices:  make(map[string]int),
	}
}

// RegisterAccount adds a new credentialed account to the pool.
func (ap *AccountPool) RegisterAccount(id, provider, apiKey string) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	h := sha256.Sum256([]byte(apiKey))
	acc := &Account{
		ID:       id,
		Provider: provider,
		KeyHash:  hex.EncodeToString(h[:8]),
		Status:   "healthy",
	}

	ap.accounts[id] = acc
	ap.byProv[provider] = append(ap.byProv[provider], acc)
}

// PickAccount selects the next available healthy account for a provider.
func (ap *AccountPool) PickAccount(provider string) (*Account, error) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	accs := ap.byProv[provider]
	if len(accs) == 0 {
		return nil, fmt.Errorf("%w for provider %s", ErrNoAvailableAccount, provider)
	}

	now := time.Now().UTC()
	startIdx := ap.indices[provider]

	for i := 0; i < len(accs); i++ {
		idx := (startIdx + i) % len(accs)
		acc := accs[idx]

		// Check if cooldown expired
		if acc.Status == "cooldown" && now.After(acc.CooldownUntil) {
			acc.Status = "healthy"
			acc.Failures = 0
		}

		if acc.Status == "healthy" {
			ap.indices[provider] = (idx + 1) % len(accs)
			acc.TotalRequests++
			return acc, nil
		}
	}

	return nil, fmt.Errorf("%w for provider %s (all accounts in cooldown/exhausted)", ErrNoAvailableAccount, provider)
}

// RecordFailure marks an account failure or sets it into cooldown.
func (ap *AccountPool) RecordFailure(accountID string, isRateLimit bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	acc := ap.accounts[accountID]
	if acc == nil {
		return
	}

	acc.Failures++
	if isRateLimit {
		acc.Status = "cooldown"
		acc.CooldownUntil = time.Now().UTC().Add(30 * time.Second)
	} else if acc.Failures >= 3 {
		acc.Status = "cooldown"
		acc.CooldownUntil = time.Now().UTC().Add(15 * time.Second)
	}
}

// RecordSuccess clears failure count on successful turn.
func (ap *AccountPool) RecordSuccess(accountID string) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	acc := ap.accounts[accountID]
	if acc != nil {
		acc.Failures = 0
		acc.Status = "healthy"
	}
}
