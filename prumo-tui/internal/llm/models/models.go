// Package models holds the model identity the client displays.
//
// It deliberately carries no catalogue. Which models exist, what they cost and
// where they run is the harness's knowledge, reached over the protocol; a
// client that shipped its own list would be asserting something it cannot
// verify. The type survives because the view layer needs a name to draw.
package models

type (
	// ModelID identifies a model as the harness names it.
	ModelID string
	// ModelProvider identifies who serves it.
	ModelProvider string
)

// Model is the model identity shown in the client.
type Model struct {
	ID       ModelID       `json:"id"`
	Name     string        `json:"name"`
	Provider ModelProvider `json:"provider"`
	// APIModel is the identifier sent upstream. The harness resolves it.
	APIModel string `json:"api_model"`

	ContextWindow    int64 `json:"context_window"`
	DefaultMaxTokens int64 `json:"default_max_tokens"`

	// Cost fields carry what the harness reported for this run, not a price
	// list the client keeps. Zero means "not reported".
	CostPer1MIn  float64 `json:"cost_per_1m_in"`
	CostPer1MOut float64 `json:"cost_per_1m_out"`

	CanReason           bool `json:"can_reason"`
	SupportsAttachments bool `json:"supports_attachments"`
}

// ProviderMock is used by the offline provider.
const ProviderMock ModelProvider = "__mock"
