package knowledge

import (
	"fmt"
	"strings"
	"unicode"

	coreknowledge "github.com/raillen/prumo/internal/knowledge"
)

// Legacy ID prefixes produced before the stable-identity migration (W3.8).
const (
	legacyRequirementPrefix = "req-"
	legacyEvidencePrefix    = "ev-"
)

// kindVocabulary maps harness record kinds to the canonical knowledge
// vocabulary. They are one vocabulary with two spellings: `open_question` is
// the harness spelling, `open-question` is the schema spelling.
var kindVocabulary = map[Kind]coreknowledge.Kind{
	KindSource:       coreknowledge.KindSource,
	KindSection:      coreknowledge.KindSource,
	KindClaim:        coreknowledge.KindClaim,
	KindFinding:      coreknowledge.KindFinding,
	KindDecision:     coreknowledge.KindDecision,
	KindRequirement:  coreknowledge.KindRequirement,
	KindConstraint:   coreknowledge.KindConstraint,
	KindRisk:         coreknowledge.KindRisk,
	KindAssumption:   coreknowledge.KindAssumption,
	KindOpenQuestion: coreknowledge.KindOpenQuestion,
	KindResearch:     coreknowledge.KindResearch,
	KindEvidence:     coreknowledge.KindEvidence,
	KindMemory:       coreknowledge.KindMemory,
}

// CanonicalKind reports the schema spelling for a harness kind.
func CanonicalKind(k Kind) (coreknowledge.Kind, bool) {
	mapped, ok := kindVocabulary[k]
	return mapped, ok
}

// Slug normalizes an arbitrary label into a valid knowledge slug:
// lower-cased, with every character outside [a-z0-9._-] folded to '-'.
func Slug(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Non-ASCII letters and digits are not part of the slug alphabet.
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-.")
	if slug == "" {
		return ""
	}
	// The identifier pattern requires an alphanumeric first character.
	if !isAlnum(slug[0]) {
		slug = "x" + slug
	}
	return slug
}

func isAlnum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

// StableID builds the durable `ku:` identifier for a record. The slug is derived
// from the record's domain identity, never from a file path or a title, so a
// rename or a move does not create a new entity (W3.7, W3.8).
func StableID(kind Kind, slug string) (string, error) {
	canonical, ok := CanonicalKind(kind)
	if !ok {
		return "", fmt.Errorf("unknown knowledge kind %q", kind)
	}
	normalized := Slug(slug)
	if normalized == "" {
		return "", fmt.Errorf("knowledge slug for kind %s is empty", kind)
	}
	id, err := coreknowledge.NewID(string(canonical) + "." + normalized)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// StableIDFor builds a stable id and falls back to a deterministic, prefixed
// slug when the label cannot be normalized at all. A record is never left
// without identity.
func StableIDFor(kind Kind, slug string) string {
	if id, err := StableID(kind, slug); err == nil {
		return id
	}
	fallback, err := coreknowledge.NewID(strings.ToLower(string(kind)) + ".unspecified")
	if err != nil {
		return string(kind) + ".unspecified"
	}
	return fallback.String()
}

// IsStableID reports whether an identifier already uses the durable scheme.
func IsStableID(id string) bool {
	_, err := coreknowledge.ParseID(id)
	return err == nil
}

// legacyAlias maps a pre-migration run-scoped identifier to the stable id the
// migration assigns it. Resolving through this function is what lets stores
// persisted before W3.8 keep working without rewriting their contents.
func legacyAlias(id string) (string, bool) {
	switch {
	case strings.HasPrefix(id, legacyRequirementPrefix):
		return StableIDFor(KindRequirement, "run-"+strings.TrimPrefix(id, legacyRequirementPrefix)), true
	case strings.HasPrefix(id, legacyEvidencePrefix):
		return StableIDFor(KindEvidence, "run-"+strings.TrimPrefix(id, legacyEvidencePrefix)), true
	}
	return "", false
}
