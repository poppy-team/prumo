// Markdown sections (GAP-007 remainder): structural editing by heading
// path — replace/insert/delete/list sections without touching the rest.
// ATX headings (#..######) define the tree; fenced code blocks are opaque
// (headings inside fences never split sections). Setext/list/table
// awareness stays future work.
package doccompile

import (
	"fmt"
	"strings"
)

// Section is one heading-addressed block.
type Section struct {
	Path  []string `json:"path"`
	Level int      `json:"level"`
	Body  string   `json:"body"`
}

type mdLine struct {
	text    string
	heading bool
	level   int
	title   string
}

func splitLinesKeep(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func parseMarkdown(doc string) []mdLine {
	raw := splitLinesKeep(doc)
	out := make([]mdLine, 0, len(raw))
	inFence := false
	for _, ln := range raw {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
		}
		m := mdLine{text: ln}
		if !inFence && strings.HasPrefix(ln, "#") {
			i := 0
			for i < len(ln) && ln[i] == '#' && i < 6 {
				i++
			}
			if i < len(ln) && ln[i] == ' ' {
				m.heading = true
				m.level = i
				m.title = strings.TrimSpace(ln[i:])
			}
		}
		out = append(out, m)
	}
	return out
}

// ListSections returns heading paths in document order.
func ListSections(doc string) [][]string {
	var out [][]string
	var stack []string
	for _, ln := range parseMarkdown(doc) {
		if !ln.heading {
			continue
		}
		for len(stack) >= ln.level {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, ln.title)
		cp := append([]string{}, stack...)
		out = append(out, cp)
	}
	return out
}

// sectionSpan locates [start,end) line indexes for a heading path.
func sectionSpan(lines []mdLine, path []string) (int, int, error) {
	if len(path) == 0 {
		return 0, 0, fmt.Errorf("empty section path")
	}
	var stack []string
	for i, ln := range lines {
		if !ln.heading {
			continue
		}
		for len(stack) >= ln.level {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, ln.title)
		if equalPath(stack, path) {
			end := len(lines)
			var inner []string
			inner = append(inner, stack...)
			for j := i + 1; j < len(lines); j++ {
				if !lines[j].heading {
					continue
				}
				for len(inner) >= lines[j].level {
					inner = inner[:len(inner)-1]
				}
				inner = append(inner, lines[j].title)
				if len(inner) <= len(path) {
					end = j
					break
				}
			}
			return i, end, nil
		}
	}
	return 0, 0, fmt.Errorf("section %q not found", strings.Join(path, " / "))
}

func equalPath(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ExtractSection returns a section including its heading line.
func ExtractSection(doc string, path []string) (string, error) {
	lines := parseMarkdown(doc)
	start, end, err := sectionSpan(lines, path)
	if err != nil {
		return "", err
	}
	texts := make([]string, 0, end-start)
	for _, ln := range lines[start:end] {
		texts = append(texts, ln.text)
	}
	return strings.Join(texts, "\n") + "\n", nil
}

// ReplaceSection swaps a section body, keeping its heading line.
func ReplaceSection(doc string, path []string, body string) (string, error) {
	lines := parseMarkdown(doc)
	start, end, err := sectionSpan(lines, path)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, ln := range lines[:start+1] {
		b.WriteString(ln.text + "\n")
	}
	b.WriteString(strings.Trim(body, "\n") + "\n")
	for _, ln := range lines[end:] {
		b.WriteString(ln.text + "\n")
	}
	return b.String(), nil
}

// DeleteSection removes a section; deleting the last remaining section is
// refused (use whole-file generation instead).
func DeleteSection(doc string, path []string) (string, error) {
	lines := parseMarkdown(doc)
	start, end, err := sectionSpan(lines, path)
	if err != nil {
		return "", err
	}
	kept := append([]mdLine{}, lines[:start]...)
	kept = append(kept, lines[end:]...)
	hasHeading := false
	for _, ln := range kept {
		if ln.heading {
			hasHeading = true
		}
	}
	if !hasHeading {
		return "", fmt.Errorf("refusing to delete the last section")
	}
	var b strings.Builder
	for _, ln := range kept {
		b.WriteString(ln.text + "\n")
	}
	return b.String(), nil
}
