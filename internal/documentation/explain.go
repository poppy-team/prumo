package docengine

import (
	"fmt"
	"sort"
	"strings"
)

// ExplainBinding resolves an explanation target: either a contract id (returns
// its bound sources) or a document path (returns the contracts that bind it).
// Unknown targets are an error, never an empty explanation (W17.12).
func ExplainBinding(root, target string) ([]string, string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, "", fmt.Errorf("explain requires a contract id or document path")
	}
	registry, err := LoadRegistry(root)
	if err != nil {
		return nil, "", err
	}
	bindings, err := LoadBindings(root)
	if err != nil {
		return nil, "", err
	}
	if _, ok := registry.Contracts[target]; ok {
		sources := []string{}
		for _, b := range bindings {
			if b.ContractID == target {
				sources = append(sources, b.Sources...)
			}
		}
		sort.Strings(sources)
		return sources, target, nil
	}
	contracts := []string{}
	for _, b := range bindings {
		for _, source := range b.Sources {
			if source == target || strings.HasPrefix(target, strings.TrimSuffix(source, "/")+"/") {
				contracts = append(contracts, b.ContractID)
				break
			}
		}
	}
	if len(contracts) == 0 {
		return nil, "", fmt.Errorf("no contract or binding matches %q", target)
	}
	sort.Strings(contracts)
	return []string{target}, strings.Join(contracts, ", "), nil
}
