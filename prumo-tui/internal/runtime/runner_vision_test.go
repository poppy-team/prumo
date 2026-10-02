package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/llm/models"
)

func TestContainsImageRef(t *testing.T) {
	cases := []struct {
		content string
		want    bool
	}{
		{"describe @shot.png", true},
		{"compare @dir/photo.JPG with @old.webp", true},
		{"look at @.prumo/cache/media/clipboard_1.png", true},
		{"see image.png for details", false},
		{"read @notes.txt first", false},
		{"no references here", false},
		{"email me@example.com", false},
	}
	for _, c := range cases {
		if got := containsImageRef(c.content); got != c.want {
			t.Errorf("containsImageRef(%q) = %v, want %v", c.content, got, c.want)
		}
	}
}

type visionStubClient struct {
	stubClient
	infos []prumo.ModelInfo
}

func (s *visionStubClient) ModelInfo(context.Context, prumo.ModelsRequest) ([]prumo.ModelInfo, error) {
	return s.infos, nil
}

func seedVisionCatalogue(t *testing.T, r *Runner, c *visionStubClient) {
	t.Helper()
	if _, _, err := r.ModelCatalogue(context.Background(), "fake"); err != nil {
		t.Fatalf("catalogue: %v", err)
	}
	_ = c
}

func TestVisionRerouteMatrix(t *testing.T) {
	newSeeded := func(t *testing.T, active models.Model, vision string) (*Runner, *visionStubClient) {
		t.Helper()
		client := &visionStubClient{infos: []prumo.ModelInfo{
			{ID: "text-only", Declared: true, Capabilities: prumo.CapabilitySet{Text: true, Tools: true}},
			{ID: "seer", Declared: true, Capabilities: prumo.CapabilitySet{Text: true, Vision: true, Tools: true}},
			{ID: "sight-4o", Declared: true, Capabilities: prumo.CapabilitySet{Text: true, Vision: true}},
		}}
		r := newTestRunner(client, active)
		seedVisionCatalogue(t, r, client)
		r.SetVisionModel(vision)
		return r, client
	}
	run := func(t *testing.T, r *Runner, c *visionStubClient, goal string) string {
		t.Helper()
		if _, err := r.Run(context.Background(), "S1", goal); err != nil {
			t.Fatalf("run: %v", err)
		}
		if len(c.started) != 1 {
			t.Fatalf("expected one start, got %d", len(c.started))
		}
		return c.started[0].Model
	}

	r, c := newSeeded(t, models.Model{ID: "text-only", Name: "text-only"}, "sight-4o")
	if got := run(t, r, c, "describe @shot.png"); got != "sight-4o" {
		t.Errorf("image goal on blind model routed to %q, want vision override", got)
	}

	r, c = newSeeded(t, models.Model{ID: "text-only", Name: "text-only"}, "sight-4o")
	if got := run(t, r, c, "refactor the parser"); got != "text-only" {
		t.Errorf("text goal rerouted to %q, want active model", got)
	}

	r, c = newSeeded(t, models.Model{ID: "seer", Name: "seer"}, "sight-4o")
	if got := run(t, r, c, "describe @shot.png"); got != "seer" {
		t.Errorf("image goal on vision model rerouted to %q, want active model", got)
	}

	r, c = newSeeded(t, models.Model{ID: "text-only", Name: "text-only"}, "")
	if got := run(t, r, c, "describe @shot.png"); got != "text-only" {
		t.Errorf("no override must keep active model, got %q", got)
	}

	r, c = newSeeded(t, models.Model{ID: "unlisted", Name: "unlisted"}, "sight-4o")
	if got := run(t, r, c, "describe @shot.png"); got != "sight-4o" {
		t.Errorf("unknown capability must fall back to override, got %q", got)
	}
}

func TestVisionModelPersistsPerWorkspace(t *testing.T) {
	ws := t.TempDir()
	client := &stubClient{}
	r := newTestRunner(client, models.Model{ID: "text-only"})
	r.SetWorkspace(ws)

	if err := r.SaveVisionModel("sight-4o"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(ws, ".prumo", "vision-model")); err != nil {
		t.Fatalf("override file missing: %v", err)
	} else if string(data) != "sight-4o\n" {
		t.Fatalf("override file = %q", data)
	}

	fresh := newTestRunner(&stubClient{}, models.Model{ID: "text-only"})
	fresh.SetWorkspace(ws)
	if got := fresh.VisionModel(); got != "sight-4o" {
		t.Fatalf("workspace load = %q, want saved override", got)
	}

	if err := r.SaveVisionModel(""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := r.VisionModel(); got != "" {
		t.Fatalf("cleared override = %q", got)
	}
	if _, err := os.Stat(filepath.Join(ws, ".prumo", "vision-model")); !os.IsNotExist(err) {
		t.Fatalf("cleared override left its file behind")
	}
}

func TestModelHasVision(t *testing.T) {
	client := &visionStubClient{infos: []prumo.ModelInfo{
		{ID: "seer", Declared: true, Capabilities: prumo.CapabilitySet{Vision: true}},
		{ID: "rumored", Declared: false, Capabilities: prumo.CapabilitySet{Vision: true}},
	}}
	r := newTestRunner(client, models.Model{ID: "text-only"})
	seedVisionCatalogue(t, r, client)

	if !r.ModelHasVision("seer") {
		t.Error("declared vision not reported")
	}
	if r.ModelHasVision("rumored") {
		t.Error("undeclared vision must not count")
	}
	if r.ModelHasVision("unknown") {
		t.Error("unknown model must not count")
	}
}
