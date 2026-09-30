package goals

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxGoalBytes = 256 << 10
	maxGoalFiles = 4096
	maxDepth     = 32
)

type Goal struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Phase       string         `json:"phase"`
	State       string         `json:"state"`
	Acceptance  []string       `json:"acceptance"`
	Constraints []string       `json:"constraints"`
	Gates       map[string]any `json:"gates"`
}

func ReadWorkspace(workspace string) []Goal {
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return nil
	}
	defer root.Close()

	var paths []string
	var walk func(string, int)
	walk = func(dir string, depth int) {
		if depth > maxDepth || len(paths) >= maxGoalFiles {
			return
		}
		entries, err := fs.ReadDir(root.FS(), dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if len(paths) >= maxGoalFiles {
				return
			}
			name := filepath.Join(dir, entry.Name())
			info, err := root.Lstat(name)
			if err != nil || info.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if info.IsDir() {
				walk(name, depth+1)
			} else if info.Mode().IsRegular() && strings.HasSuffix(entry.Name(), ".goal.json") {
				paths = append(paths, name)
			}
		}
	}
	walk(filepath.Join(".ai", "goals"), 0)
	sort.Strings(paths)

	result := make([]Goal, 0, len(paths))
	for _, path := range paths {
		info, err := root.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxGoalBytes {
			continue
		}
		file, err := root.Open(path)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(file, maxGoalBytes+1))
		file.Close()
		if err != nil || len(data) > maxGoalBytes {
			continue
		}
		var goal Goal
		if json.Unmarshal(data, &goal) != nil || strings.TrimSpace(goal.ID) == "" {
			continue
		}
		if goal.Title == "" {
			goal.Title = goal.ID
		}
		goal.State = strings.ToUpper(goal.State)
		result = append(result, goal)
	}
	return result
}
