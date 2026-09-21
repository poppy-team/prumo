package guitest

import (
	"strings"
)

// Locator defines the canonical Atlas common locator model for native GUI elements per Chapter 26.
type Locator struct {
	Role         string `json:"role,omitempty"`
	Name         string `json:"name,omitempty"`
	AutomationID string `json:"automation_id,omitempty"`
	Label        string `json:"label,omitempty"`
	Value        string `json:"value,omitempty"`
	Description  string `json:"description,omitempty"`
	State        string `json:"state,omitempty"`
	Parent       string `json:"parent,omitempty"`
	Relation     string `json:"relation,omitempty"`
}

// UIElement represents an accessible UI element snapshot in the native accessibility tree.
type UIElement struct {
	ID           string            `json:"id"`
	Role         string            `json:"role"`
	Name         string            `json:"name"`
	AutomationID string            `json:"automation_id"`
	Label        string            `json:"label"`
	Value        string            `json:"value"`
	Description  string            `json:"description"`
	State        string            `json:"state"`
	Visible      bool              `json:"visible"`
	Focused      bool              `json:"focused"`
	Enabled      bool              `json:"enabled"`
	Children     []*UIElement      `json:"children,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// Matches checks if a UI element matches all non-empty criteria in the locator.
func (l *Locator) Matches(el *UIElement) bool {
	if el == nil {
		return false
	}
	if l.Role != "" && !strings.EqualFold(el.Role, l.Role) {
		return false
	}
	if l.Name != "" && !strings.Contains(strings.ToLower(el.Name), strings.ToLower(l.Name)) {
		return false
	}
	if l.AutomationID != "" && el.AutomationID != l.AutomationID {
		return false
	}
	if l.Label != "" && !strings.Contains(strings.ToLower(el.Label), strings.ToLower(l.Label)) {
		return false
	}
	if l.Value != "" && el.Value != l.Value {
		return false
	}
	if l.Description != "" && !strings.Contains(strings.ToLower(el.Description), strings.ToLower(l.Description)) {
		return false
	}
	if l.State != "" && !strings.EqualFold(el.State, l.State) {
		return false
	}
	return true
}

// FindFirst searches a UIElement tree depth-first for the first matching element.
func (l *Locator) FindFirst(root *UIElement) *UIElement {
	if root == nil {
		return nil
	}
	if l.Matches(root) {
		return root
	}
	for _, child := range root.Children {
		if found := l.FindFirst(child); found != nil {
			return found
		}
	}
	return nil
}
