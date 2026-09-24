package runtime

import (
	"encoding/base64"
	"fmt"
	"github.com/raillen/prumo/internal/harness/safepath"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
)

// MaxImageBudget is the aggregate limit for all image references in one request
// (32MB). MaxImageSize bounds one image; this bounds the request, because eight
// images under the per-image cap are still eight times the budget anyone set.
const MaxImageBudget = 32 * 1024 * 1024

// MaxImageSize is the upper limit for an image reference (10MB).
// Files exceeding this limit are rejected with an explicit error rather than
// sent truncated.
const MaxImageSize = 10 * 1024 * 1024

var (
	refRegex = regexp.MustCompile(`@([^\s,;:"'<>\(\)\[\]\{\}]+)`)

	imageMimeTypes = map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".webp": "image/webp",
		".gif":  "image/gif",
	}
)

// IsImageReference reports whether a path has an image extension recognized by
// the harness.
func IsImageReference(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := imageMimeTypes[ext]
	return ok
}

// ImageMimeType returns the MIME type for an image path, or empty if unrecognized.
func ImageMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return imageMimeTypes[ext]
}

type refMatch struct {
	start int
	end   int
	path  string
	full  string
}

// ResolveReferences scans user messages for @image references.
//
// When the model declares vision, referenced image files are read, encoded to
// base64, and attached as ContentPart blocks alongside the text. If the model
// does not declare vision, references remain plain text in the prompt.
// Files exceeding MaxImageSize are rejected with a clear error.
func ResolveReferences(msgs []agent.Message, workspace string, hasVision bool) ([]agent.Message, error) {
	out := make([]agent.Message, len(msgs))
	copy(out, msgs)

	for i, m := range out {
		if m.Role != agent.RoleUser {
			continue
		}
		if len(m.Parts) > 0 {
			// Already partitioned into parts
			continue
		}
		if !hasVision {
			// Models that do not declare vision receive the prompt as plain text.
			continue
		}

		matches := refRegex.FindAllStringSubmatchIndex(m.Content, -1)
		if len(matches) == 0 {
			continue
		}

		var validRefs []refMatch
		var attachedBytes int64
		for _, idx := range matches {
			fullStart, fullEnd := idx[0], idx[1]
			pathStart, pathEnd := idx[2], idx[3]
			path := m.Content[pathStart:pathEnd]
			if !IsImageReference(path) {
				continue
			}

			// The reference is resolved inside the workspace, and a reference that
			// leaves it is not an image — it is a path. "@../../etc/passwd" joined
			// without containment reads whatever the agent can reach and inlines
			// it into a model request. Absolute paths were accepted for the same
			// reason. An image reference that points outside the run's workspace
			// means the run is being pointed at something it was not given
			// (GAP-113).
			target, err := safepath.Resolve(workspace, path, true)
			if err != nil {
				// Not readable, not a file, or outside the workspace. Either way it
				// is not an image this run may attach, so the reference stays
				// plain text — which is also what the model sees if it asks.
				continue
			}
			fi, err := os.Stat(target)
			if err != nil || fi.IsDir() {
				// File does not exist on disk: leave reference as plain text
				continue
			}

			if fi.Size() > MaxImageSize {
				return nil, fmt.Errorf("image reference @%s exceeds 10MB limit (%d bytes)", path, fi.Size())
			}
			// The per-image limit was never an aggregate. Eight 9MB images are
			// eight times the budget anyone set, and the cap that exists gives the
			// impression the request is bounded when it is not.
			attachedBytes += fi.Size()
			if attachedBytes > MaxImageBudget {
				return nil, fmt.Errorf("image references exceed the %dMB request budget (%d bytes across %d images)",
					MaxImageBudget/(1<<20), attachedBytes, len(validRefs)+1)
			}

			validRefs = append(validRefs, refMatch{
				start: fullStart,
				end:   fullEnd,
				path:  target,
				full:  path,
			})
		}

		if len(validRefs) == 0 {
			continue
		}

		var parts []agent.ContentPart
		lastIdx := 0

		for _, ref := range validRefs {
			if ref.start > lastIdx {
				textChunk := m.Content[lastIdx:ref.start]
				if textChunk != "" {
					parts = append(parts, agent.ContentPart{Type: "text", Text: textChunk})
				}
			}

			data, err := os.ReadFile(ref.path)
			if err != nil {
				return nil, fmt.Errorf("reading image @%s: %w", ref.full, err)
			}

			parts = append(parts, agent.ContentPart{
				Type:     "image",
				MimeType: ImageMimeType(ref.path),
				Data:     base64.StdEncoding.EncodeToString(data),
				Path:     ref.full,
			})

			lastIdx = ref.end
		}

		if lastIdx < len(m.Content) {
			trailing := m.Content[lastIdx:]
			if trailing != "" {
				parts = append(parts, agent.ContentPart{Type: "text", Text: trailing})
			}
		}

		out[i].Parts = parts
	}

	return out, nil
}
