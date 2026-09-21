package guitest

import (
	"context"
	"fmt"
	"time"
)

// Scenario represents an Atlas desktop GUI test scenario.
type Scenario struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	App             AppConfig `json:"app"`
	PreferredDriver string    `json:"preferred_driver,omitempty"`
	Steps           []Step    `json:"steps"`
}

// Step represents a single interaction or assertion step in a GUI scenario.
type Step struct {
	Action       string        `json:"action"` // launch, find, click, double_click, type, shortcut, focus, wait_for, assert, screenshot, close
	Locator      *Locator      `json:"locator,omitempty"`
	Input        string        `json:"input,omitempty"`
	Expectations *Expectation  `json:"expectations,omitempty"`
	TimeoutMs    int           `json:"timeout_ms,omitempty"`
}

// Expectation represents asserted conditions on an element.
type Expectation struct {
	Visible *bool   `json:"visible,omitempty"`
	Focused *bool   `json:"focused,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	Text    string  `json:"text,omitempty"`
	Value   string  `json:"value,omitempty"`
	State   string  `json:"state,omitempty"`
}

// ActionTimelineEntry records each action executed during a scenario run.
type ActionTimelineEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Target    string    `json:"target,omitempty"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	Duration  string    `json:"duration"`
}

// RunResult encapsulates the outcome of a scenario execution per Chapter 26 evidence rules.
type RunResult struct {
	ScenarioID            string                `json:"scenario_id"`
	Passed                bool                  `json:"passed"`
	ErrorMessage          string                `json:"error_message,omitempty"`
	FailedStepIndex       int                   `json:"failed_step_index,omitempty"`
	ActionTimeline        []ActionTimelineEntry `json:"action_timeline"`
	AccessibilitySnapshot *UIElement            `json:"accessibility_snapshot,omitempty"`
	ScreenshotBytes       []byte                `json:"-"`
	Duration              time.Duration         `json:"duration"`
}

// Runner executes Atlas desktop GUI test scenarios against a Driver.
type Runner struct {
	driver Driver
}

// NewRunner creates a new Runner with the specified Driver.
func NewRunner(driver Driver) *Runner {
	if driver == nil {
		driver = NewMockDriver(nil)
	}
	return &Runner{driver: driver}
}

// Run executes the scenario steps sequentially.
func (r *Runner) Run(ctx context.Context, s Scenario) (*RunResult, error) {
	startTime := time.Now()
	res := &RunResult{
		ScenarioID:     s.ID,
		Passed:         true,
		ActionTimeline: make([]ActionTimelineEntry, 0, len(s.Steps)),
	}

	var session Session
	defer func() {
		if session != nil {
			_ = session.Close(context.Background())
		}
		res.Duration = time.Since(startTime)
	}()

	for i, step := range s.Steps {
		stepStart := time.Now()
		entry := ActionTimelineEntry{
			Timestamp: stepStart,
			Action:    step.Action,
		}
		if step.Locator != nil {
			entry.Target = fmt.Sprintf("%+v", *step.Locator)
		}

		err := r.executeStep(ctx, step, &session, s.App, res)
		entry.Duration = time.Since(stepStart).String()

		if err != nil {
			entry.Success = false
			entry.Error = err.Error()
			res.ActionTimeline = append(res.ActionTimeline, entry)
			res.Passed = false
			res.ErrorMessage = fmt.Sprintf("step %d (%s) failed: %v", i, step.Action, err)
			res.FailedStepIndex = i

			if session != nil {
				if tree, treeErr := session.GetAccessibilityTree(ctx); treeErr == nil {
					res.AccessibilitySnapshot = tree
				}
				if shot, shotErr := session.CaptureScreenshot(ctx); shotErr == nil {
					res.ScreenshotBytes = shot
				}
			}
			return res, nil
		}

		entry.Success = true
		res.ActionTimeline = append(res.ActionTimeline, entry)
	}

	if session != nil {
		if tree, treeErr := session.GetAccessibilityTree(ctx); treeErr == nil {
			res.AccessibilitySnapshot = tree
		}
		if shot, shotErr := session.CaptureScreenshot(ctx); shotErr == nil {
			res.ScreenshotBytes = shot
		}
	}

	return res, nil
}

func (r *Runner) executeStep(ctx context.Context, step Step, sess *Session, appConfig AppConfig, res *RunResult) error {
	switch step.Action {
	case "launch":
		s, err := r.driver.Launch(ctx, appConfig)
		if err != nil {
			return fmt.Errorf("launch failed: %w", err)
		}
		*sess = s
		return nil

	case "close":
		if *sess != nil {
			err := (*sess).Close(ctx)
			*sess = nil
			return err
		}
		return nil

	case "find":
		if *sess == nil {
			return fmt.Errorf("no active session; did you call launch?")
		}
		if step.Locator == nil {
			return fmt.Errorf("find requires a locator")
		}
		_, err := (*sess).FindElement(ctx, *step.Locator)
		return err

	case "click":
		if *sess == nil {
			return fmt.Errorf("no active session")
		}
		if step.Locator == nil {
			return fmt.Errorf("click requires a locator")
		}
		return (*sess).Click(ctx, *step.Locator)

	case "double_click":
		if *sess == nil {
			return fmt.Errorf("no active session")
		}
		if step.Locator == nil {
			return fmt.Errorf("double_click requires a locator")
		}
		return (*sess).DoubleClick(ctx, *step.Locator)

	case "type":
		if *sess == nil {
			return fmt.Errorf("no active session")
		}
		if step.Locator == nil {
			return fmt.Errorf("type requires a locator")
		}
		return (*sess).TypeText(ctx, *step.Locator, step.Input)

	case "shortcut":
		if *sess == nil {
			return fmt.Errorf("no active session")
		}
		return (*sess).SendShortcut(ctx, step.Input)

	case "assert":
		if *sess == nil {
			return fmt.Errorf("no active session")
		}
		if step.Locator == nil {
			return fmt.Errorf("assert requires a locator")
		}
		el, err := (*sess).FindElement(ctx, *step.Locator)
		if err != nil {
			return fmt.Errorf("assert failed to find target: %w", err)
		}
		if step.Expectations != nil {
			exp := step.Expectations
			if exp.Visible != nil && el.Visible != *exp.Visible {
				return fmt.Errorf("expected visible=%v, got %v", *exp.Visible, el.Visible)
			}
			if exp.Enabled != nil && el.Enabled != *exp.Enabled {
				return fmt.Errorf("expected enabled=%v, got %v", *exp.Enabled, el.Enabled)
			}
			if exp.Focused != nil && el.Focused != *exp.Focused {
				return fmt.Errorf("expected focused=%v, got %v", *exp.Focused, el.Focused)
			}
			if exp.Value != "" && el.Value != exp.Value {
				return fmt.Errorf("expected value=%q, got %q", exp.Value, el.Value)
			}
			if exp.Text != "" && el.Name != exp.Text {
				return fmt.Errorf("expected text=%q, got %q", exp.Text, el.Name)
			}
		}
		return nil

	case "screenshot":
		if *sess == nil {
			return fmt.Errorf("no active session")
		}
		bytes, err := (*sess).CaptureScreenshot(ctx)
		if err != nil {
			return err
		}
		res.ScreenshotBytes = bytes
		return nil

	default:
		return fmt.Errorf("unsupported action %q", step.Action)
	}
}
