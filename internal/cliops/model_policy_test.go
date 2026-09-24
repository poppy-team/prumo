package cliops

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/resolver"
	"github.com/raillen/prumo/internal/validation"
)

// The writer emitted each role as {"preferred": [...], "fallback": [...]}; the
// doctor expected each role to be the name of a profile and did `continue` on
// anything else — which is every role the writer produces. The role check had
// therefore never examined a single file the framework itself generated, and
// reported no problems by skipping the work (GAP-132).

func findingMessages(findings []map[string]any) string {
	messages := make([]string, 0, len(findings))
	for _, finding := range findings {
		messages = append(messages, finding["message"].(string))
	}
	return strings.Join(messages, "\n")
}

func hasFinding(findings []map[string]any, substring string) bool {
	return strings.Contains(findingMessages(findings), substring)
}

// generatedPolicyForTest is the shape Init writes, taken from the writer itself
// so the test cannot drift from it.
func generatedPolicyForTest(t *testing.T) map[string]any {
	t.Helper()
	svc := New(repoRoot(t))
	policy, err := svc.ModelPolicy(profileWithOneModel())
	if err != nil {
		t.Fatal(err)
	}
	return roundTrip(t, policy)
}

// roundTrip marshals and decodes, because the doctor reads a file: the checks
// are written against decoded JSON, where a list is []any rather than the
// []string the writer holds in memory.
func roundTrip(t *testing.T, value map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func profileWithOneModel() resolver.Profile {
	return resolver.Profile{Raw: map[string]any{
		"ai": map[string]any{
			"preferred_models": []any{
				map[string]any{"id": "m1", "provider": "prov-a"},
			},
		},
	}}
}

func loadModelPolicySchema(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "schemas", "model-policy.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func validateAgainstSchema(t *testing.T, schema map[string]any, policy map[string]any) error {
	t.Helper()
	// The policy is checked as decoded JSON, which is how the doctor will read it
	// from disk, rather than as the Go value the writer holds.
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	registry, err := validation.LoadRegistry(filepath.Join(repoRoot(t), "schemas"))
	if err != nil {
		return err
	}
	if messages := validation.ValidateSchema(decoded, schema, registry); len(messages) > 0 {
		return errors.New(strings.Join(messages, "; "))
	}
	return nil
}

func TestTheGeneratedPolicyIsActuallyCheckedByTheDoctor(t *testing.T) {
	policy := generatedPolicyForTest(t)
	// The doctor must read every role in the file it is given, not skip the
	// ones it does not recognise. A policy with no findings here means the roles
	// were parsed, not that they were passed over — which the next test proves.
	if findings := modelPolicyFindings(policy); len(findings) != 0 {
		t.Fatalf("the generated policy must be clean, got:\n%s", findingMessages(findings))
	}
}

func TestARoleTheDoctorCannotUnderstandIsAnError(t *testing.T) {
	// Before, an unparseable role was `continue`d and reported as fine. The
	// silent skip is the failure mode: a policy the doctor cannot read is a
	// policy the doctor cannot check.
	policy := map[string]any{
		"roster": []any{map[string]any{"id": "m1"}},
		"roles":  map[string]any{"architect": true},
	}
	findings := modelPolicyFindings(policy)
	if len(findings) == 0 {
		t.Fatal("a role that is neither a profile name nor a model list must be reported")
	}
	if !hasFinding(findings, "architect") {
		t.Fatalf("the finding must name the role, got:\n%s", findingMessages(findings))
	}
}

func TestARoleThatNamesAModelOutsideTheRosterIsReported(t *testing.T) {
	// Nothing checked this anywhere. A role naming a model the project never
	// configured surfaced as a routing failure at run time instead of as a bad
	// policy at compile time.
	policy := map[string]any{
		"roster": []any{map[string]any{"id": "m1"}},
		"roles": map[string]any{
			"architect": map[string]any{"preferred": []any{"m1", "typo-model"}},
		},
	}
	findings := modelPolicyFindings(policy)
	if !hasFinding(findings, "typo-model") {
		t.Fatalf("a model id outside the roster must be reported, got:\n%s", findingMessages(findings))
	}
}

func TestAFallbackModelOutsideTheRosterIsAlsoReported(t *testing.T) {
	policy := map[string]any{
		"roster": []any{map[string]any{"id": "m1"}},
		"roles": map[string]any{
			"reviewer": map[string]any{
				"preferred": []any{"m1"},
				"fallback":  []any{"also-not-configured"},
			},
		},
	}
	if findings := modelPolicyFindings(policy); !hasFinding(findings, "also-not-configured") {
		t.Fatalf("a fallback outside the roster must be reported, got:\n%s", findingMessages(findings))
	}
}

func TestTheProfileFormStillWorksAndIsChecked(t *testing.T) {
	policy := map[string]any{
		"roster":   []any{map[string]any{"id": "m1"}},
		"profiles": map[string]any{"fast": map[string]any{"provider": "p", "model": "m"}},
		"roles":    map[string]any{"architect": "fast", "reviewer": "missing-profile"},
	}
	findings := modelPolicyFindings(policy)
	// The check names the profile the role referenced, not the role, so the
	// assertion has to look for the profile rather than for a word that happens
	// to be shared between a role name and a profile name.
	if hasFinding(findings, `references missing profile "fast"`) {
		t.Fatalf("a profile that exists must not be reported, got:\n%s", findingMessages(findings))
	}
	if !hasFinding(findings, `references missing profile "missing-profile"`) {
		t.Fatalf("a role naming a profile that does not exist must be reported, got:\n%s", findingMessages(findings))
	}
}

func TestAnUnreadablePolicyIsReportedRatherThanTreatedAsClean(t *testing.T) {
	// A file whose roles object is missing entirely: reporting "no findings"
	// about a policy that cannot be read is the false green this whole change is
	// about.
	findings := modelPolicyFindings(map[string]any{"roster": []any{map[string]any{"id": "m1"}}})
	if len(findings) == 0 {
		t.Fatal("a policy with no roles must be reported")
	}
}

func TestADuplicateRosterIdIsReported(t *testing.T) {
	// With the same id twice, "which model does this role mean" is ambiguous and
	// the resolver would pick one without saying so.
	policy := map[string]any{
		"roster": []any{map[string]any{"id": "m1"}, map[string]any{"id": "m1"}},
		"roles":  map[string]any{"architect": map[string]any{"preferred": []any{"m1"}}},
	}
	if findings := modelPolicyFindings(policy); !hasFinding(findings, "more than once") {
		t.Fatalf("a duplicate roster id must be reported, got:\n%s", findingMessages(findings))
	}
}

func TestARoleThatSelectsNothingIsReported(t *testing.T) {
	policy := map[string]any{
		"roster": []any{map[string]any{"id": "m1"}},
		"roles":  map[string]any{"architect": map[string]any{}},
	}
	if findings := modelPolicyFindings(policy); !hasFinding(findings, "selects nothing") {
		t.Fatalf("a role with no model ids must be reported, got:\n%s", findingMessages(findings))
	}
}

func TestTheSchemaAcceptsTheGeneratedShape(t *testing.T) {
	// The schema pinned nothing about role values, so it passed the writer's
	// shape and the reader's shape alike while they disagreed. If the generated
	// policy fails its own schema, the contract is still not expressed.
	schema := loadModelPolicySchema(t)
	policy := generatedPolicyForTest(t)
	if err := validateAgainstSchema(t, schema, policy); err != nil {
		t.Fatalf("the generated policy must satisfy its own schema: %v", err)
	}
}

func TestTheSchemaRejectsARoleShapeNeitherSideUses(t *testing.T) {
	schema := loadModelPolicySchema(t)
	policy := map[string]any{
		"version":               2,
		"selection_rule":        "x",
		"cross_provider_review": true,
		"roster":                []any{map[string]any{"id": "m1", "provider": "p"}},
		"roles":                 map[string]any{"architect": 42},
	}
	if err := validateAgainstSchema(t, schema, policy); err == nil {
		t.Fatal("the schema must reject a role value that is neither a profile name nor a model list")
	}
}

func TestTheSchemaRequiresARoleToSelectSomething(t *testing.T) {
	schema := loadModelPolicySchema(t)
	policy := map[string]any{
		"version":               2,
		"selection_rule":        "x",
		"cross_provider_review": true,
		"roster":                []any{map[string]any{"id": "m1", "provider": "p"}},
		"roles":                 map[string]any{"architect": map[string]any{}},
	}
	if err := validateAgainstSchema(t, schema, policy); err == nil {
		t.Fatal("a role that selects no model must fail the schema")
	}
}
