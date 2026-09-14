// Token estimation table (GAP-027): documented heuristic, not measurement.
// Default 4.0 chars/token; per-model overrides stay conservative (higher
// estimate) so budgets err toward safety. Measured calibration is future
// work; the table version is pinned for auditability.
package model

import (
	"io"
	"net/http"
	"strings"
)

// EstimatorVersion pins the heuristic revision.
const EstimatorVersion = "tokens-v1"

// charsPerToken holds conservative chars/token ratios by model substring.
var charsPerToken = []struct {
	substr string
	ratio  float64
}{
	{"claude", 3.5},
	{"gpt", 4.0},
	{"gemini", 4.0},
	{"llama", 3.5},
	{"mistral", 3.5},
	{"default", 4.0},
}

// EstimateTokens returns a conservative token estimate for text.
func EstimateTokens(text, modelName string) int {
	ratio := 4.0
	lower := strings.ToLower(modelName)
	for _, e := range charsPerToken {
		if e.substr == "default" {
			continue
		}
		if strings.Contains(lower, e.substr) {
			ratio = e.ratio
			break
		}
	}
	n := int(float64(len(text)) / ratio)
	if n < 1 {
		n = 1
	}
	return n
}

// readAll reads a bounded discovery payload.
func readAll(resp *http.Response) []byte {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return data
}
func CappedEstimate(text, modelName string, cap int) int {
	n := EstimateTokens(text, modelName)
	if n > cap {
		n = cap
	}
	return n
}
