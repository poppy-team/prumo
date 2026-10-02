package docengine

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	prumo "github.com/raillen/prumo"
)

type Contract struct {
	ID                   string         `json:"id"`
	Version              int            `json:"version"`
	Role                 string         `json:"role"`
	Description          string         `json:"description"`
	Applicability        map[string]any `json:"applicability"`
	RequiredKnowledge    []string       `json:"required_knowledge"`
	BlockingQuestions    []string       `json:"blocking_questions"`
	EvidenceRequirements []string       `json:"evidence_requirements"`
	UpdateTriggers       []string       `json:"update_triggers"`
	EligibleOwners       []string       `json:"eligible_owners"`
	Staleness            struct {
		Mode string `json:"mode"`
	} `json:"staleness"`
}

type Profile struct {
	ID                string   `json:"id"`
	Version           int      `json:"version"`
	Description       string   `json:"description"`
	Capabilities      []string `json:"capabilities"`
	Extends           []string `json:"extends"`
	Contracts         []string `json:"contracts"`
	ExcludedContracts []string `json:"excluded_contracts"`
}

type Registry struct {
	Contracts map[string]Contract
	Profiles  map[string]Profile
}

func LoadRegistry(root string) (Registry, error) {
	registry := Registry{Contracts: map[string]Contract{}, Profiles: map[string]Profile{}}
	contractsPath := filepath.Join(root, "docs", "contracts", "builtin.json")
	profilesPath := filepath.Join(root, "docs", "profiles", "builtin.json")

	var contractsData, profilesData []byte

	// 1. Project-local overrides win if present.
	if _, err := os.Stat(contractsPath); err == nil {
		contractsData, _ = os.ReadFile(contractsPath)
		profilesData, _ = os.ReadFile(profilesPath)
	}

	// 2. An explicit environment override allows testing custom registries.
	if len(contractsData) == 0 {
		if envRoot := os.Getenv("PRUMO_REPO_ROOT"); envRoot != "" {
			candContracts := filepath.Join(envRoot, "docs", "contracts", "builtin.json")
			candProfiles := filepath.Join(envRoot, "docs", "profiles", "builtin.json")
			if _, err := os.Stat(candContracts); err == nil {
				contractsData, _ = os.ReadFile(candContracts)
				profilesData, _ = os.ReadFile(candProfiles)
			}
		}
	}

	// 3. The embedded framework registry is authoritative across any environment.
	// A hardcoded personal developer path was deleted here: shipping a personal
	// home path in production code meant the control plane only worked on the
	// original author's machine.
	if len(contractsData) == 0 {
		cFS := prumo.EmbeddedDocContracts()
		pFS := prumo.EmbeddedDocProfiles()
		if data, err := fs.ReadFile(cFS, "builtin.json"); err == nil {
			contractsData = data
		}
		if data, err := fs.ReadFile(pFS, "builtin.json"); err == nil {
			profilesData = data
		}
	}

	if len(contractsData) == 0 {
		return registry, fmt.Errorf("documentation contract registry not found; expected docs/contracts/builtin.json or embedded registry")
	}

	var contracts []Contract
	if err := json.Unmarshal(contractsData, &contracts); err != nil {
		return registry, fmt.Errorf("parse documentation contracts: %w", err)
	}
	for _, contract := range contracts {
		if contract.ID == "" || contract.Version < 1 {
			return registry, fmt.Errorf("invalid documentation contract")
		}
		if _, exists := registry.Contracts[contract.ID]; exists {
			return registry, fmt.Errorf("duplicate documentation contract: %s", contract.ID)
		}
		registry.Contracts[contract.ID] = contract
	}
	if len(profilesData) == 0 {
		return registry, fmt.Errorf("documentation profile registry not found; expected docs/profiles/builtin.json or embedded registry")
	}
	var profiles []Profile
	if err := json.Unmarshal(profilesData, &profiles); err != nil {
		return registry, fmt.Errorf("parse documentation profiles: %w", err)
	}
	for _, profile := range profiles {
		if profile.ID == "" || profile.Version < 1 {
			return registry, fmt.Errorf("invalid documentation profile")
		}
		if _, exists := registry.Profiles[profile.ID]; exists {
			return registry, fmt.Errorf("duplicate documentation profile: %s", profile.ID)
		}
		registry.Profiles[profile.ID] = profile
	}
	return registry, nil
}

func (r Registry) ResolveProfiles(profileIDs []string, capabilities map[string]bool) ([]Contract, []string, error) {
	selected := map[string]bool{}
	visiting := map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("profile composition cycle: %s", id)
		}
		profile, ok := r.Profiles[id]
		if !ok {
			return fmt.Errorf("unknown documentation profile: %s", id)
		}
		if selected["profile:"+id] {
			return nil
		}
		visiting[id] = true
		for _, parent := range profile.Extends {
			if err := visit(parent); err != nil {
				return err
			}
		}
		for _, contractID := range profile.Contracts {
			selected[contractID] = true
		}
		for _, excluded := range profile.ExcludedContracts {
			delete(selected, excluded)
		}
		visiting[id] = false
		selected["profile:"+id] = true
		return nil
	}
	for _, id := range profileIDs {
		if err := visit(id); err != nil {
			return nil, nil, err
		}
	}
	ids := []string{}
	for id := range selected {
		if len(id) >= 8 && id[:8] == "profile:" {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	contracts := []Contract{}
	unknown := []string{}
	for _, id := range ids {
		contract, ok := r.Contracts[id]
		if !ok {
			unknown = append(unknown, id)
			continue
		}
		if Applicable(contract, capabilities) {
			contracts = append(contracts, contract)
		}
	}
	return contracts, unknown, nil
}

func Applicable(contract Contract, capabilities map[string]bool) bool {
	if contract.Applicability == nil {
		return true
	}
	if raw, ok := contract.Applicability["capabilities_any"].([]any); ok {
		for _, value := range raw {
			if capabilities[fmt.Sprint(value)] {
				return true
			}
		}
		return false
	}
	return true
}
