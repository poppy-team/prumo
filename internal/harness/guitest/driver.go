package guitest

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// Standard Provider Driver IDs per Chapter 26
const (
	DriverWindowsUIA = "gui.windows-uia"
	DriverMacOSAX    = "gui.macos-ax"
	DriverLinuxATSPI = "gui.linux-atspi"
	DriverMacOSXCUI  = "gui.macos-xcui"
	DriverQtSquish   = "gui.qt-squish"
	DriverMock       = "mock"
)

// Capabilities represents the capability negotiation matrix for a driver.
type Capabilities struct {
	SupportsSemanticUI        bool `json:"supports_semantic_ui"`
	SupportsInteraction       bool `json:"supports_interaction"`
	SupportsVisualScreenshots bool `json:"supports_visual_screenshots"`
	SupportsAccessibilityTree bool `json:"supports_accessibility_tree"`
	SupportsClipboard         bool `json:"supports_clipboard"`
	SupportsDragAndDrop       bool `json:"supports_drag_and_drop"`
}

// Session represents an active GUI application test session.
type Session interface {
	GetAccessibilityTree(ctx context.Context) (*UIElement, error)
	FindElement(ctx context.Context, loc Locator) (*UIElement, error)
	Click(ctx context.Context, loc Locator) error
	DoubleClick(ctx context.Context, loc Locator) error
	TypeText(ctx context.Context, loc Locator, text string) error
	SendShortcut(ctx context.Context, combo string) error
	CaptureScreenshot(ctx context.Context) ([]byte, error)
	Close(ctx context.Context) error
}

// Driver defines the interface that all platform GUI drivers must implement.
type Driver interface {
	DriverID() string
	Capabilities() Capabilities
	Launch(ctx context.Context, config AppConfig) (Session, error)
}

// AppConfig defines execution parameters for the GUI target application.
type AppConfig struct {
	Executable string            `json:"executable"`
	Args       []string          `json:"args"`
	Cwd        string            `json:"cwd"`
	Env        map[string]string `json:"env"`
	Timeout    time.Duration     `json:"timeout"`
}

// DetectPlatformDefaultDriver returns the default driver ID for the current OS.
func DetectPlatformDefaultDriver() string {
	switch runtime.GOOS {
	case "windows":
		return DriverWindowsUIA
	case "darwin":
		return DriverMacOSAX
	case "linux":
		return DriverLinuxATSPI
	default:
		return DriverMock
	}
}

// MockDriver provides an in-memory, deterministic implementation for CI and tests.
type MockDriver struct {
	RootElement *UIElement
}

func NewMockDriver(root *UIElement) *MockDriver {
	if root == nil {
		root = &UIElement{
			ID:   "root",
			Role: "window",
			Name: "Prumo Native",
			Children: []*UIElement{
				{
					ID:           "editor",
					Role:         "edit",
					AutomationID: "code_editor",
					Name:         "Code Editor",
					Visible:      true,
					Focused:      true,
					Enabled:      true,
					Value:        "initial code",
				},
				{
					ID:           "btn_save",
					Role:         "button",
					AutomationID: "save_btn",
					Name:         "Save (Ctrl+S)",
					Visible:      true,
					Enabled:      true,
				},
			},
		}
	}
	return &MockDriver{RootElement: root}
}

func (m *MockDriver) DriverID() string {
	return DriverMock
}

func (m *MockDriver) Capabilities() Capabilities {
	return Capabilities{
		SupportsSemanticUI:        true,
		SupportsInteraction:       true,
		SupportsVisualScreenshots: true,
		SupportsAccessibilityTree: true,
		SupportsClipboard:         true,
		SupportsDragAndDrop:       false,
	}
}

func (m *MockDriver) Launch(ctx context.Context, config AppConfig) (Session, error) {
	return &mockSession{
		root: m.RootElement,
	}, nil
}

type mockSession struct {
	root *UIElement
}

func (s *mockSession) GetAccessibilityTree(ctx context.Context) (*UIElement, error) {
	return s.root, nil
}

func (s *mockSession) FindElement(ctx context.Context, loc Locator) (*UIElement, error) {
	el := loc.FindFirst(s.root)
	if el == nil {
		return nil, fmt.Errorf("element matching locator not found: %+v", loc)
	}
	return el, nil
}

func (s *mockSession) Click(ctx context.Context, loc Locator) error {
	_, err := s.FindElement(ctx, loc)
	return err
}

func (s *mockSession) DoubleClick(ctx context.Context, loc Locator) error {
	_, err := s.FindElement(ctx, loc)
	return err
}

func (s *mockSession) TypeText(ctx context.Context, loc Locator, text string) error {
	el, err := s.FindElement(ctx, loc)
	if err != nil {
		return err
	}
	el.Value = text
	return nil
}

func (s *mockSession) SendShortcut(ctx context.Context, combo string) error {
	return nil
}

func (s *mockSession) CaptureScreenshot(ctx context.Context) ([]byte, error) {
	// 1x1 dummy PNG bytes
	return []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82"), nil
}

func (s *mockSession) Close(ctx context.Context) error {
	return nil
}
