package guitest

import (
	"context"
	"testing"
)

func TestLocatorMatchingAndTreeLookup(t *testing.T) {
	root := &UIElement{
		ID:   "root",
		Role: "window",
		Name: "Prumo Native Workspace Viewer",
		Children: []*UIElement{
			{
				ID:           "sidebar",
				Role:         "panel",
				AutomationID: "explorer_panel",
				Name:         "Explorer",
				Children: []*UIElement{
					{
						ID:    "file_item",
						Role:  "treeitem",
						Name:  "src/main.rs",
						Label: "src/main.rs",
						State: "selected",
					},
				},
			},
			{
				ID:           "editor",
				Role:         "edit",
				AutomationID: "code_editor",
				Name:         "Editor",
				Visible:      true,
				Focused:      true,
				Enabled:      true,
				Value:        "fn main() {}",
			},
		},
	}

	// Match by AutomationID
	locAutoID := Locator{AutomationID: "code_editor"}
	el1 := locAutoID.FindFirst(root)
	if el1 == nil || el1.ID != "editor" {
		t.Fatalf("Expected to find editor by automation ID, got %v", el1)
	}

	// Match by Role + Name partial
	locTreeItem := Locator{Role: "treeitem", Name: "main.rs"}
	el2 := locTreeItem.FindFirst(root)
	if el2 == nil || el2.ID != "file_item" {
		t.Fatalf("Expected to find file item by role and name, got %v", el2)
	}

	// Non-matching locator
	locNotFound := Locator{Name: "nonexistent"}
	if el3 := locNotFound.FindFirst(root); el3 != nil {
		t.Fatalf("Expected nil for nonexistent element, got %v", el3)
	}
}

func TestScenarioRunnerExecution(t *testing.T) {
	driver := NewMockDriver(nil)
	runner := NewRunner(driver)

	visible := true
	enabled := true
	scenario := Scenario{
		ID:   "gui:test-edit-save",
		Name: "Test edit code and assert save",
		App: AppConfig{
			Executable: "prumo-native",
		},
		Steps: []Step{
			{Action: "launch"},
			{
				Action: "assert",
				Locator: &Locator{
					AutomationID: "code_editor",
				},
				Expectations: &Expectation{
					Visible: &visible,
					Enabled: &enabled,
					Value:   "initial code",
				},
			},
			{
				Action: "type",
				Locator: &Locator{
					AutomationID: "code_editor",
				},
				Input: "modified code by test",
			},
			{
				Action: "assert",
				Locator: &Locator{
					AutomationID: "code_editor",
				},
				Expectations: &Expectation{
					Value: "modified code by test",
				},
			},
			{
				Action: "click",
				Locator: &Locator{
					AutomationID: "save_btn",
				},
			},
			{
				Action: "screenshot",
			},
			{
				Action: "close",
			},
		},
	}

	ctx := context.Background()
	res, err := runner.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Runner.Run returned unexpected Go error: %v", err)
	}
	if !res.Passed {
		t.Fatalf("Scenario run failed: %s (at step %d)", res.ErrorMessage, res.FailedStepIndex)
	}
	if len(res.ActionTimeline) != len(scenario.Steps) {
		t.Fatalf("Expected %d timeline entries, got %d", len(scenario.Steps), len(res.ActionTimeline))
	}
	if len(res.ScreenshotBytes) == 0 {
		t.Fatalf("Expected screenshot bytes to be populated")
	}
}

func TestScenarioRunnerAssertionFailure(t *testing.T) {
	driver := NewMockDriver(nil)
	runner := NewRunner(driver)

	scenario := Scenario{
		ID:   "gui:test-failing-assert",
		Name: "Test failing assertion",
		App: AppConfig{
			Executable: "prumo-native",
		},
		Steps: []Step{
			{Action: "launch"},
			{
				Action: "assert",
				Locator: &Locator{
					AutomationID: "code_editor",
				},
				Expectations: &Expectation{
					Value: "expected wrong content",
				},
			},
		},
	}

	ctx := context.Background()
	res, err := runner.Run(ctx, scenario)
	if err != nil {
		t.Fatalf("Expected nil Go error, got %v", err)
	}
	if res.Passed {
		t.Fatalf("Expected scenario to fail assertion, but it passed")
	}
	if res.FailedStepIndex != 1 {
		t.Fatalf("Expected failed step index 1, got %d", res.FailedStepIndex)
	}
}

func TestDoctorCheckDiagnostics(t *testing.T) {
	ctx := context.Background()
	findings := DoctorCheck(ctx)
	if len(findings) == 0 {
		t.Fatalf("Expected at least session/accessibility diagnostic findings")
	}
	hasSession := false
	hasAccessibility := false
	for _, f := range findings {
		if f.Category == "gui/session" {
			hasSession = true
		}
		if f.Category == "gui/accessibility" {
			hasAccessibility = true
		}
	}
	if !hasSession || !hasAccessibility {
		t.Fatalf("Expected both gui/session and gui/accessibility categories, got %v", findings)
	}
}

func TestDriverCapabilityNegotiation(t *testing.T) {
	driver := NewMockDriver(nil)
	caps := driver.Capabilities()
	if !caps.SupportsSemanticUI || !caps.SupportsInteraction || !caps.SupportsVisualScreenshots {
		t.Fatalf("Expected mock driver to support Semantic UI, Interaction and Visual, got %+v", caps)
	}
	defaultDriverID := DetectPlatformDefaultDriver()
	if defaultDriverID == "" {
		t.Fatalf("Expected default driver ID to be detected")
	}
}
