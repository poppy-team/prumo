package install

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// GeneratedMarker identifies a file this framework wrote.
//
// It exists because ownership cannot be inferred from a path. AGENTS.md,
// CLAUDE.md and GEMINI.md are conventions that belong to the user first: an
// existing one is theirs, and a generated one is ours. Without a marker in the
// content, the only way to tell them apart at uninstall time is a manifest that
// a previous version may have written while clobbering the user's file — which is
// exactly the state where deleting is most destructive.
const GeneratedMarker = "prumo:managed"

// ErrFileBelongsToSomeoneElse reports that a write was declined because the file
// already exists and is not ours.
//
// It is a distinct value rather than a plain error so a caller can tell "could not
// write" from "declined to destroy something". The first is a failure; the second
// is a decision, and the right response to it is to carry on without claiming
// ownership of the file.
var ErrFileBelongsToSomeoneElse = errors.New("install: a file this framework did not write already exists")

// Skipped records a file this framework declined to touch, and why.
type Skipped struct {
	Path   string
	Reason string
}

// WriteResult says what happened to one path.
type WriteResult struct {
	// Path is the file considered.
	Path string
	// Written is true when the file is now the content passed in.
	Written bool
	// Owned is true when this framework may delete the file later.
	//
	// It is false for a pre-existing file that was left alone, and a false here
	// is what keeps an uninstall from deleting somebody's work.
	Owned bool
	// Skipped explains a refusal, when there was one.
	Skipped *Skipped
}

// WriteManaged writes content to path unless doing so would destroy a file this
// framework did not write.
//
// The three outcomes are deliberately distinct, because collapsing them is the
// bug:
//
//   - The file does not exist: write it, and own it.
//   - The file exists and is byte-identical, or carries our marker: refresh it,
//     and own it. A file we can identify is safe to update.
//   - The file exists, differs, and has no marker: leave it alone and say so.
//     Not owning it is what means the uninstall will not remove it.
//
// The third case is the whole point. Codex, Claude Code, Gemini and Antigravity
// are told to read AGENTS.md, CLAUDE.md and GEMINI.md from the project root —
// files users write themselves, usually with project instructions in them. Writing
// over one destroyed those instructions, and listing the path as created meant
// the uninstall deleted the file outright, so the loss was permanent and
// unrecoverable (GAP-141).
func WriteManaged(path, content string) (WriteResult, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return WriteResult{Path: path}, err
	}
	if err == nil && len(existing) > 0 && !bytes.Equal(existing, []byte(content)) && !IsManaged(existing) {
		return WriteResult{
			Path: path, Written: false, Owned: false,
			Skipped: &Skipped{Path: path, Reason: "a file this framework did not write already exists here"},
		}, nil
	}
	if err := writeFile(path, content); err != nil {
		return WriteResult{Path: path}, err
	}
	return WriteResult{Path: path, Written: true, Owned: true}, nil
}

// looksLikeJSON reports whether content is a JSON document, which cannot carry a
// comment marker without becoming unreadable.
func looksLikeJSON(content string) bool {
	trimmed := strings.TrimLeft(content, " \t\r\n")
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

// WriteRecord writes a file that records this framework's own ownership, always.
//
// It exists because the general rule is wrong for these. A rule that permits
// overwriting a file only when the new bytes match the old ones is self-defeating
// for an ownership manifest, whose entire purpose is to change as the set of
// created files changes: the first run and the second produce different lists, so
// the second run refused to update the record of what it had created and the
// install failed on its own bookkeeping.
//
// These paths are inside the connector's own dot-directory, are named for us, and
// are this framework's by definition — so the question of stealing a user's file
// does not arise for them.
//
// It is deliberately not the path used for AGENTS.md and friends. Those are the
// user's, and the whole point of the marker is that we can tell the difference.
func WriteRecord(path, content string) (WriteResult, error) {
	if err := writeFile(path, content); err != nil {
		return WriteResult{Path: path}, err
	}
	return WriteResult{Path: path, Written: true, Owned: true}, nil
}

// IsManaged reports whether content is a file this framework generated.
func IsManaged(content []byte) bool {
	return bytes.Contains(content, []byte(GeneratedMarker))
}

// writeFile writes through a temp file and renames, so a failure partway does not
// leave a half-written instructions file where an agent will read it.
func writeFile(path, content string) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Marked appends the managed marker to generated content, so ownership is
// detectable from the file itself rather than only from a manifest.
//
// It deliberately leaves JSON alone. The marker is an HTML comment, which is
// legal in Markdown and in nothing else: putting it in a .json file produces a
// document no parser will read, and the first version of this did exactly that,
// failing connector validation with "invalid JSON" on every generated file.
//
// JSON artifacts cannot carry the marker, so their ownership rests on the
// cleanup manifest and on the fact that they live inside the connector's own
// dot-directory. The files that need the marker are the ones in the project root
// that the user owns by convention, and those are Markdown.
func Marked(content string) string {
	if IsManaged([]byte(content)) {
		return content
	}
	if looksLikeJSON(content) {
		return content
	}
	var b strings.Builder
	b.WriteString("<!-- ")
	b.WriteString(GeneratedMarker)
	b.WriteString(" -->\n")
	b.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

// RemoveManagedPaths removes the paths this framework created, refusing any it
// cannot prove it owns.
//
// home is what makes the refusal decidable, and the distinction is the whole
// point. Two directories are in play and they belong to different parties:
//
//   - Under home, everything is this framework's own state: its cache, its
//     manifests, its checkpoints. A manifest naming a path there is authoritative,
//     because nothing else writes there.
//   - Under the project, a file belongs to the user unless it carries the managed
//     marker. AGENTS.md, CLAUDE.md and GEMINI.md live in the project root and are
//     the user's by convention; a manifest from an older version may name one,
//     and deleting on the strength of that record destroys work with no way back
//     (GAP-141).
//
// So a path is removed when it is under home, when it is recognisably marked as
// ours, or when it is empty. Anything else is reported as a leftover and left
// exactly where it is.
//
// The cost of being wrong in the refusing direction is a file the user deletes
// themselves. The cost of being wrong the other way is their instructions.
func RemoveManagedPaths(home string, paths []string) (removed []string, leftovers []string) {
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if !ownedForRemoval(home, path) {
			leftovers = append(leftovers, path)
			continue
		}
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil || len(entries) > 0 {
				// A populated directory is never removed. os.Remove would fail on
				// it anyway, which means the old branch could only ever leave a
				// confusing error behind.
				leftovers = append(leftovers, path)
				continue
			}
			if err := os.Remove(path); err != nil {
				leftovers = append(leftovers, path)
				continue
			}
			removed = append(removed, path)
			continue
		}
		if err := os.Remove(path); err != nil {
			leftovers = append(leftovers, path)
			continue
		}
		removed = append(removed, path)
	}
	return removed, leftovers
}

// FrameworkOwnedDirs are the directories a connector owns outright.
//
// A file inside one of these is ours by namespace: the names are ours, we created
// the directories, and a manifest naming a file inside one is authoritative. They
// are listed rather than inferred from "has a directory component", because a
// project can have any directory it likes and treating all of them as ours would
// make the refusal meaningless.
var FrameworkOwnedDirs = []string{
	".codex", ".claude", ".gemini", ".antigravity", ".opencode", ".agents", ".prumo",
}

// ownedForRemoval reports whether a path is safe to delete.
//
// A file sitting directly in the project root is never ours unless it carries the
// marker. That is the case the whole rule exists for: AGENTS.md, CLAUDE.md and
// GEMINI.md are conventions the user owns, they are never inside a framework
// directory, and an older manifest may well name one.
func ownedForRemoval(home, path string) bool {
	if home != "" {
		if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	if insideFrameworkDir(path) {
		return true
	}
	content, err := os.ReadFile(path)
	if err != nil {
		// Unreadable means unprovable, and unprovable means refused.
		return false
	}
	// An empty file carries no content to destroy.
	return len(content) == 0 || IsManaged(content)
}

// insideFrameworkDir reports whether path sits inside a directory this framework
// owns. A root-level file never qualifies, which is the point.
func insideFrameworkDir(path string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	if len(parts) < 2 {
		return false
	}
	// Drop the file name, then look for a framework directory among the parents.
	for _, part := range parts[:len(parts)-1] {
		for _, owned := range FrameworkOwnedDirs {
			if part == owned {
				return true
			}
		}
	}
	return false
}

// RequireOwnership reports paths in a created list that this framework cannot
// prove it owns, for a caller that wants to say so before deleting anything.
func RequireOwnership(home string, paths []string) (owned, foreign []string) {
	for _, path := range paths {
		if ownedForRemoval(home, path) {
			owned = append(owned, path)
			continue
		}
		foreign = append(foreign, path)
	}
	return owned, foreign
}
