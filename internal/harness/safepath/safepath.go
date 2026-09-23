// Package safepath is the single containment boundary for every path the
// harness resolves and every identifier it turns into a path.
//
// Before it existed, containment was reimplemented four times — in
// harness/aci, toolgateway, harness/directive and environment — and all four
// were lexical: they compared `filepath.Rel` results without ever resolving a
// symbolic link. A symlink inside the workspace pointing at /etc therefore
// passed every check, which is the escape the security document claims is
// rejected (GAP-110).
//
// Identifiers had no boundary at all. A daemon run id was interpolated straight
// into record, event, diff, checkpoint and context paths, so a value containing
// a separator reached outside the store (GAP-112). ValidateID is the check that
// closes it.
//
// One boundary, one set of rules: Resolve for filesystem paths, ValidateID for
// anything that becomes a path component.
package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve returns the absolute path for p confined to root.
//
// The containment test is done twice: lexically, so the obvious `..` escape is
// rejected with a clear error, and again after resolving symbolic links, so a
// link inside the root cannot lead outside it. Both are needed. The lexical
// check alone misses symlinks; the symlink check alone cannot classify a
// malformed path or one that does not exist yet.
//
// A path that does not exist is resolved as far as it can: the deepest existing
// ancestor is resolved and the remainder appended. That is what lets edit.create
// confine a new file inside a root where some parent component is a symlink.
//
// mustExist selects strict behaviour. When true a path that does not exist is
// an error; when false, a missing leaf is allowed as long as its existing
// ancestors stay inside the root.
func Resolve(root, p string, mustExist bool) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("safepath: empty root")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("safepath: resolve root: %w", err)
	}
	absRoot = filepath.Clean(absRoot)
	// Resolve the root itself: it may itself sit behind a symlink, and comparing
	// a resolved candidate against an unresolved root reports a false escape.
	if resolvedRoot, evalErr := filepath.EvalSymlinks(absRoot); evalErr == nil {
		absRoot = resolvedRoot
	}

	candidate := p
	if candidate == "" {
		candidate = "."
	}
	abs := candidate
	if !filepath.IsAbs(candidate) {
		abs = filepath.Join(absRoot, candidate)
	}
	abs = filepath.Clean(abs)

	// Lexical containment, before touching the filesystem. Cheap, and it turns
	// the common escape into an error that says what happened.
	if !contains(absRoot, abs) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}

	info, statErr := os.Lstat(abs)
	switch {
	case statErr == nil && info.Mode()&os.ModeSymlink != 0:
		// The candidate is itself a link: where does it actually point?
		target, evalErr := filepath.EvalSymlinks(abs)
		if evalErr != nil {
			return "", fmt.Errorf("resolve symlink %s: %w", p, evalErr)
		}
		if !contains(absRoot, target) {
			return "", fmt.Errorf("path escapes workspace through a symlink: %s -> %s", p, target)
		}
		return target, nil
	case statErr == nil:
		// The candidate exists and is not a link. A parent component can still
		// be one, so resolve the deepest existing ancestor and re-check.
		resolvedParent, evalErr := filepath.EvalSymlinks(abs)
		if evalErr != nil {
			return "", fmt.Errorf("resolve %s: %w", p, evalErr)
		}
		if !contains(absRoot, resolvedParent) {
			return "", fmt.Errorf("path escapes workspace through a symlink: %s -> %s", p, resolvedParent)
		}
		return resolvedParent, nil
	case mustExist:
		return "", fmt.Errorf("resolve %s: %w", p, statErr)
	default:
		// Missing path: resolve the deepest existing ancestor, confirm it is
		// inside the root, then re-append the components that do not exist yet.
		resolved, remaining, err := resolveDeepestExisting(absRoot, abs)
		if err != nil {
			return "", err
		}
		if remaining == "" {
			return resolved, nil
		}
		full := filepath.Join(resolved, remaining)
		if !contains(absRoot, full) {
			return "", fmt.Errorf("path escapes workspace: %s", p)
		}
		return full, nil
	}
}

// resolveDeepestExisting walks up from candidate until it finds something that
// exists, resolves it, and returns it with the unresolved tail.
func resolveDeepestExisting(root, candidate string) (resolved, remaining string, err error) {
	current := candidate
	var tail []string
	for {
		if _, statErr := os.Lstat(current); statErr == nil {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", "", fmt.Errorf("no existing ancestor for %s", candidate)
		}
		tail = append([]string{filepath.Base(current)}, tail...)
		current = parent
	}
	resolvedCurrent, evalErr := filepath.EvalSymlinks(current)
	if evalErr != nil {
		return "", "", fmt.Errorf("resolve ancestor of %s: %w", candidate, evalErr)
	}
	if !contains(root, resolvedCurrent) {
		return "", "", fmt.Errorf("path escapes workspace through a symlink: %s", candidate)
	}
	return resolvedCurrent, filepath.Join(tail...), nil
}

// contains reports whether candidate is root or lives beneath it.
//
// Both are compared cleaned. A prefix comparison on the raw strings would accept
// "/repo-evil" for root "/repo", which is a sibling rather than a child.
func contains(root, candidate string) bool {
	if candidate == root {
		return true
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	// A leading separator would mean Rel produced an absolute path, which can
	// only happen for a mismatch.
	return !filepath.IsAbs(rel)
}

// IsWithin reports whether candidate is root or beneath it, without touching the
// filesystem. Use it when the path may not exist yet and no symlink resolution
// is wanted; use Resolve when the destination will be read or written.
func IsWithin(root, candidate string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	abs := candidate
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(absRoot, abs)
	}
	return contains(filepath.Clean(absRoot), filepath.Clean(abs))
}

// ValidateID checks that value is safe to use as a single path component.
//
// This is the check that was missing for run ids, context manifest ids,
// checkpoint ids and adoption action names. A value containing a separator, a
// parent reference, a drive letter or a NUL could otherwise escape the store it
// was meant to live in (GAP-112).
//
// The permitted set is intentionally narrow: letters, digits, dot, underscore
// and hyphen. It is the set the JSON schemas already use for their id patterns,
// so a valid id in a document is a valid id here.
func ValidateID(kind, value string) error {
	if value == "" {
		return fmt.Errorf("%s is empty", kind)
	}
	if len(value) > 200 {
		return fmt.Errorf("%s is longer than 200 bytes: %d", kind, len(value))
	}
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s contains a NUL byte", kind)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%s is a path reference: %q", kind, value)
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return fmt.Errorf("%s contains an unsupported character %q: %s", kind, r, value)
		}
	}
	// A leading dot is how a hidden file is spelled; an id is not one.
	if strings.HasPrefix(value, ".") {
		return fmt.Errorf("%s must not start with a dot: %s", kind, value)
	}
	return nil
}

// ValidateIDs checks a whole set and reports every offender, so a caller fixing a
// batch of records learns about all of them at once.
func ValidateIDs(kind string, values []string) error {
	var problems []string
	for _, value := range values {
		if err := ValidateID(kind, value); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d invalid %s value(s): %s", len(problems), kind, strings.Join(problems, "; "))
	}
	return nil
}
