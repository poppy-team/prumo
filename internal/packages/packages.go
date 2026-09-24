package packages

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type Lock struct {
	Version  int        `json:"version"`
	Packages []Resolved `json:"packages"`
}
type Resolved struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
	Type     string `json:"type"`
}

func NewLock(packages []Resolved) Lock {
	sort.Slice(packages, func(i, j int) bool { return packages[i].ID < packages[j].ID })
	return Lock{Version: 1, Packages: packages}
}
func Checksum(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// ChecksumTree is a stable checksum of a whole directory.
//
// The per-file Checksum is what a lock entry used to record, and it was taken
// over manifest.json alone: the lock then said a package was the version that
// was asked for while its script, reference or SKILL.md had changed. A lock that
// cannot see most of what it is locking does not lock (GAP-147).
//
// The digest covers each file's relative path and its bytes, so a file that
// moves is a change, and the result does not depend on the order the filesystem
// happens to return directory entries.
func ChecksumTree(root string) (string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)

	hasher := sha256.New()
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		// The path is part of the digest: two trees that differ only in where a
		// file lives are different packages.
		hasher.Write([]byte(filepath.ToSlash(rel)))
		hasher.Write([]byte{0})
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		hasher.Write(data)
		hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
