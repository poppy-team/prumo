package runtime

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
)

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
		for _, idx := range matches {
			fullStart, fullEnd := idx[0], idx[1]
			pathStart, pathEnd := idx[2], idx[3]
			path := m.Content[pathStart:pathEnd]
			if !IsImageReference(path) {
				continue
			}

			target := path
			if !filepath.IsAbs(target) && workspace != "" {
				target = filepath.Join(workspace, path)
			}
			target = filepath.Clean(target)

			fi, err := os.Stat(target)
			if err != nil || fi.IsDir() {
				// File does not exist on disk: leave reference as plain text
				continue
			}

			if fi.Size() > MaxImageSize {
				return nil, fmt.Errorf("image reference @%s exceeds 10MB limit (%d bytes)", path, fi.Size())
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
