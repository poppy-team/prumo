package runtime

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

func TestResolveReferencesWithoutVision(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "screen.png")
	if err := os.WriteFile(imgPath, []byte("fake-png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	msgs := []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "what is wrong with @screen.png?"},
	}

	// Model does not declare vision -> reference remains plain text
	resolved, err := ResolveReferences(msgs, dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved[0].Parts) != 0 {
		t.Fatalf("model without vision must not attach parts, got %d", len(resolved[0].Parts))
	}
}

func TestResolveReferencesWithVision(t *testing.T) {
	dir := t.TempDir()
	imgBytes := []byte("image-data-png")
	imgPath := filepath.Join(dir, "screen.png")
	if err := os.WriteFile(imgPath, imgBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	msgs := []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "check @screen.png carefully"},
	}

	resolved, err := ResolveReferences(msgs, dir, true)
	if err != nil {
		t.Fatalf("resolution failed: %v", err)
	}

	parts := resolved[0].Parts
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts (text, image, text), got %d: %+v", len(parts), parts)
	}

	if parts[0].Type != "text" || parts[0].Text != "check " {
		t.Errorf("part 0 mismatch: %+v", parts[0])
	}
	if parts[1].Type != "image" || parts[1].MimeType != "image/png" || parts[1].Path != "screen.png" {
		t.Errorf("part 1 mismatch: %+v", parts[1])
	}
	if parts[1].Data != base64.StdEncoding.EncodeToString(imgBytes) {
		t.Errorf("part 1 data mismatch: %s", parts[1].Data)
	}
	if parts[2].Type != "text" || parts[2].Text != " carefully" {
		t.Errorf("part 2 mismatch: %+v", parts[2])
	}
}

func TestResolveReferencesMissingFile(t *testing.T) {
	dir := t.TempDir()
	msgs := []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "look at @missing.png"},
	}

	resolved, err := ResolveReferences(msgs, dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolved[0].Parts) != 0 {
		t.Fatalf("missing file must not produce parts, got: %+v", resolved[0].Parts)
	}
}

func TestResolveReferencesExceedsSizeLimit(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "large.jpg")

	// Create a sparse/large file exceeding 10MB
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxImageSize + 1024); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	msgs := []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "process @large.jpg"},
	}

	_, err = ResolveReferences(msgs, dir, true)
	if err == nil {
		t.Fatal("expected size limit error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds 10MB limit") {
		t.Fatalf("expected 10MB limit message, got: %v", err)
	}
}

func TestResolveReferencesMultiple(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.png"), []byte("png1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "two.jpeg"), []byte("jpeg2"), 0o644); err != nil {
		t.Fatal(err)
	}

	msgs := []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "compare @one.png with @two.jpeg now"},
	}

	resolved, err := ResolveReferences(msgs, dir, true)
	if err != nil {
		t.Fatal(err)
	}

	parts := resolved[0].Parts
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d: %+v", len(parts), parts)
	}
	if parts[1].Type != "image" || parts[1].MimeType != "image/png" {
		t.Errorf("part 1 mismatch: %+v", parts[1])
	}
	if parts[3].Type != "image" || parts[3].MimeType != "image/jpeg" {
		t.Errorf("part 3 mismatch: %+v", parts[3])
	}
}

func TestRunnerStepResolvesVisionReferences(t *testing.T) {
	dir := t.TempDir()
	imgBytes := []byte("img-content")
	if err := os.WriteFile(filepath.Join(dir, "diag.png"), imgBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	var capturedReq agent.ModelRequest
	fakeModel := &capturingModel{
		onStream: func(req agent.ModelRequest) {
			capturedReq = req
		},
	}

	runner := NewRunner(Services{
		Models:    fakeModel,
		Workspace: dir,
		HasVision: true,
	}, "R-vision", "S-1")

	runner.SeedMessages([]agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "analyze @diag.png"},
	})

	ctx := context.Background()
	// Step PhasePrepare -> PhaseCompileContext
	if err := runner.Step(ctx); err != nil {
		t.Fatal(err)
	}
	// Step PhaseCompileContext -> PhaseRequestModel
	if err := runner.Step(ctx); err != nil {
		t.Fatal(err)
	}
	// Step PhaseRequestModel -> streams
	if err := runner.Step(ctx); err != nil {
		t.Fatal(err)
	}

	if len(capturedReq.Messages) == 0 {
		t.Fatal("no messages captured")
	}
	parts := capturedReq.Messages[0].Parts
	if len(parts) < 2 {
		t.Fatalf("expected captured message to have image parts, got: %+v", parts)
	}
	if parts[1].Type != "image" || parts[1].Path != "diag.png" {
		t.Fatalf("expected image part for diag.png, got: %+v", parts[1])
	}
}

type capturingModel struct {
	onStream func(req agent.ModelRequest)
}

func (c *capturingModel) Name() string { return "capturing" }
func (c *capturingModel) Capabilities() model.Capabilities {
	return model.Capabilities{Streaming: true}
}
func (c *capturingModel) Models(context.Context) ([]string, error) { return []string{"test"}, nil }
func (c *capturingModel) Health(context.Context) (string, error)   { return "healthy", nil }
func (c *capturingModel) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	if c.onStream != nil {
		c.onStream(req)
	}
	ch := make(chan agent.ModelEvent, 1)
	ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}
	close(ch)
	return ch, nil
}
