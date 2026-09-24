package cliops

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// model-policy.schema.json and the code that reads and writes
// .ai/orchestration/model-policy.json are one contract, and the two halves of the
// code disagreed about it. The writer emitted each role as
// {"preferred": [...], "fallback": [...]}; the reader expected each role to be
// the name of a profile and did `continue` on anything else — which is every
// role the writer produces. The role check therefore never ran on a single file
// the framework itself generated, and passed by skipping (GAP-132).
//
// The shape is defined once here and both sides go through it. The schema
// permits two forms, because a hand-written policy may name a profile per role
// while the generated one lists model ids per role; what matters is that both
// forms are understood and checked rather than one being silently ignored.

// RolePolicy is one role's model selection.
//
// A role either names a profile, or lists the roster ids it prefers and falls
// back to. Naming a profile is a convenience for a hand-written policy; listing
// ids is what the generator emits, because the profile indirection buys nothing
// when every role gets the same roster.
type RolePolicy struct {
	// Profile names an entry in the policy's profiles. Set only in the string
	// form; empty in the list form.
	Profile string
	// Preferred is the roster ids this role wants, most preferred first.
	Preferred []string
	// Fallback is the roster ids to try when none of Preferred works.
	Fallback []string
}

// usesProfileForm reports whether this role names a profile rather than listing
// model ids.
func (r RolePolicy) usesProfileForm() bool { return r.Profile != "" }

// parseRolePolicy reads both accepted role forms.
//
// The bare-string form is the profile name. Anything else must be an object with
// a list of model ids. A value that is neither is an error naming what was
// found, rather than a value dropped on the floor: a role the reader cannot
// understand is a role the reader cannot check, and a policy that cannot be
// checked is a policy that is decorative.
func parseRolePolicy(role string, raw any) (RolePolicy, error) {
	if name, ok := raw.(string); ok {
		if name == "" {
			return RolePolicy{}, fmt.Errorf("role %q is an empty profile name", role)
		}
		return RolePolicy{Profile: name}, nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return RolePolicy{}, fmt.Errorf("role %q is a %s; expected a profile name or an object with preferred and fallback model ids", role, jsonTypeName(raw))
	}
	policy := RolePolicy{}
	for _, field := range []string{"preferred", "fallback"} {
		value, present := obj[field]
		if !present {
			continue
		}
		ids, err := stringList(role, field, value)
		if err != nil {
			return RolePolicy{}, err
		}
		if field == "preferred" {
			policy.Preferred = ids
		} else {
			policy.Fallback = ids
		}
	}
	if policy.Preferred == nil && policy.Fallback == nil {
		return RolePolicy{}, fmt.Errorf("role %q has neither preferred nor fallback model ids; a role that selects nothing will be routed to whatever is cheapest", role)
	}
	return policy, nil
}

func stringList(role, field string, value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("role %q field %q is a %s; expected a list of model ids", role, field, jsonTypeName(value))
	}
	out := make([]string, 0, len(items))
	for i, item := range items {
		id, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("role %q field %q entry %d is a %s; expected a model id", role, field, i, jsonTypeName(item))
		}
		out = append(out, id)
	}
	return out, nil
}

// jsonTypeName names a decoded JSON value's type for an error message. Saying
// "is a boolean" is the difference between a report someone can act on and one
// they have to go read the file to decode.
func jsonTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "a boolean"
	case float64:
		return "a number"
	case string:
		return "a string"
	case []any:
		return "a list"
	case map[string]any:
		return "an object"
	default:
		return fmt.Sprintf("%T", value)
	}
}

// modelPolicy is the parsed .ai/orchestration/model-policy.json.
type modelPolicy struct {
	// RosterIDs are the model ids the policy declares available.
	RosterIDs []string
	// Profiles are the named profiles, empty when the policy uses the list form.
	Profiles map[string]any
	// Roles is every role with its selection.
	Roles map[string]RolePolicy
}

// parseModelPolicy reads the whole file.
//
// A file this cannot understand is an error, not an empty policy. Returning
// zero roles would let the caller report "no problems found" about a file whose
// every role was unreadable.
func parseModelPolicy(policy map[string]any) (modelPolicy, error) {
	parsed := modelPolicy{Profiles: map[string]any{}}

	if raw, ok := policy["profiles"].(map[string]any); ok {
		parsed.Profiles = raw
	}
	seenRoster := map[string]bool{}
	if raw, ok := policy["roster"].([]any); ok {
		for i, item := range raw {
			entry, ok := item.(map[string]any)
			if !ok {
				return modelPolicy{}, fmt.Errorf("roster entry %d is a %s; expected an object with an id", i, jsonTypeName(item))
			}
			id, ok := entry["id"].(string)
			if !ok || id == "" {
				return modelPolicy{}, fmt.Errorf("roster entry %d has no string id", i)
			}
			if seenRoster[id] {
				// A duplicate id makes "which model does this role mean" ambiguous,
				// and the resolver would pick one without saying so.
				return modelPolicy{}, fmt.Errorf("roster lists %q more than once", id)
			}
			seenRoster[id] = true
			parsed.RosterIDs = append(parsed.RosterIDs, id)
		}
	}

	rawRoles, ok := policy["roles"].(map[string]any)
	if !ok {
		return modelPolicy{}, fmt.Errorf("policy has no roles object")
	}
	parsed.Roles = make(map[string]RolePolicy, len(rawRoles))
	for role, raw := range rawRoles {
		rolePolicy, err := parseRolePolicy(role, raw)
		if err != nil {
			return modelPolicy{}, err
		}
		parsed.Roles[role] = rolePolicy
	}
	return parsed, nil
}

// modelPolicyFindings reports what is wrong with a parsed policy, as Doctor
// findings.
//
// The check that matters and did not exist: a role's model ids must exist in the
// roster. Nothing anywhere verified it, so a role could name a model the project
// never configured and the error would surface as a routing failure at run time
// rather than as a bad policy at compile time.
func modelPolicyFindings(policy map[string]any) []map[string]any {
	const target = ".ai/orchestration/model-policy.json"
	parsed, err := parseModelPolicy(policy)
	if err != nil {
		return []map[string]any{{
			"category": "model-policy", "severity": "ERROR",
			"message": err.Error(), "target": target,
		}}
	}
	roster := make(map[string]bool, len(parsed.RosterIDs))
	for _, id := range parsed.RosterIDs {
		roster[id] = true
	}
	var findings []map[string]any
	roles := make([]string, 0, len(parsed.Roles))
	for role := range parsed.Roles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		rolePolicy := parsed.Roles[role]
		if rolePolicy.usesProfileForm() {
			if _, ok := parsed.Profiles[rolePolicy.Profile]; !ok {
				findings = append(findings, map[string]any{
					"category": "model-policy", "severity": "ERROR",
					"message": fmt.Sprintf("Role %q references missing profile %q", role, rolePolicy.Profile),
					"target":  target,
				})
			}
			continue
		}
		for _, id := range append(append([]string{}, rolePolicy.Preferred...), rolePolicy.Fallback...) {
			if !roster[id] {
				findings = append(findings, map[string]any{
					"category": "model-policy", "severity": "ERROR",
					"message": fmt.Sprintf("Role %q references model %q, which is not in the roster", role, id),
					"target":  target,
				})
			}
		}
		if len(rolePolicy.Preferred) == 0 {
			findings = append(findings, map[string]any{
				"category": "model-policy", "severity": "WARN",
				"message": fmt.Sprintf("Role %q has fallback models but no preferred model; it will route to fallback first", role),
				"target":  target,
			})
		}
	}
	return findings
}

// renderRolePolicy is the inverse of parseRolePolicy: it writes the form the
// parser accepts. Both directions live here so a change to one is a change to
// the other, which is the whole point of defining the contract once.
func renderRolePolicy(policy RolePolicy) (any, error) {
	if policy.usesProfileForm() {
		return policy.Profile, nil
	}
	out := map[string]any{}
	if policy.Preferred != nil {
		out["preferred"] = policy.Preferred
	}
	if policy.Fallback != nil {
		out["fallback"] = policy.Fallback
	}
	return out, nil
}

// modelPolicyFindingMessage summarises findings for an error at generation time,
// where there is no file to point at yet.
func modelPolicyFindingMessage(findings []map[string]any) string {
	messages := make([]string, 0, len(findings))
	for _, finding := range findings {
		message, _ := finding["message"].(string)
		messages = append(messages, message)
	}
	return strings.Join(messages, "; ")
}

// validateRenderedModelPolicy checks a policy after the marshalling the doctor
// will actually read back, so the check runs on the same representation it is
// meant to police.
func validateRenderedModelPolicy(policy map[string]any) error {
	data, err := marshalPythonCompatible(policy)
	if err != nil {
		return fmt.Errorf("generated model policy could not be marshalled: %w", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("generated model policy does not decode: %w", err)
	}
	if findings := modelPolicyFindings(decoded); len(findings) > 0 {
		return fmt.Errorf("generated model policy is invalid: %s", modelPolicyFindingMessage(findings))
	}
	return nil
}
