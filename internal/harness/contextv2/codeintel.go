package contextv2

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/raillen/prumo/internal/harness/lsp"
	"github.com/raillen/prumo/internal/harness/repomap"
)

// codeIntelItems adds a repo-map pointer plus top symbol hits. The map is
// pointer-sized (render-capped); symbols carry file:line pointers, never
// full bodies. Failures degrade to no items — never fail compilation.
func codeIntelItems(root, goal string) []Item {
	idx := repomap.Build(root, 200, 2000, 1<<16)
	var out []Item
	rendered := idx.Render(1500)
	if rendered != "" {
		cost := estimateTokens(rendered)
		if cost < 50 {
			// Floor: pointer-sized maps must not outrank the goal on
			// utility-per-token arithmetic alone.
			cost = 50
		}
		out = append(out, Item{
			Ref: "repo-map", Authority: "reference", Trust: "medium",
			Privacy: "internal", Freshness: "current",
			Score: 0.55, Method: "repo-map", TokenCost: cost,
			Content: rendered,
		})
	}
	lang := ""
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		lang = "go"
	}
	prov := lsp.Select(root, lang, idx)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	syms, err := prov.WorkspaceSymbols(ctx, goal)
	if err != nil {
		return out
	}
	for i, s := range syms {
		if i >= 5 {
			break
		}
		out = append(out, Item{
			Ref: "symbol:" + s.File, Authority: "reference", Trust: "medium",
			Privacy: "internal", Freshness: "current",
			Score: 0.7 - 0.05*float64(i), Method: "symbols",
			TokenCost: 20, Content: s.Kind + " " + s.Name + " (" + s.Provider + ")",
		})
	}
	return out
}
