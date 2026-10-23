package codegraph

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// Meta returns the build metadata stored in the database.
func (s *Store) Meta() (Meta, error) {
	var m Meta
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bMeta)
		if b == nil {
			return errors.New("graph database is empty; run `graph build` first")
		}
		raw := b.Get(keyInfo)
		if raw == nil {
			return nil
		}
		return json.Unmarshal(raw, &m)
	})
	return m, err
}

// Stale reports whether a Go source file (or go.mod/go.sum) under the build directory was
// modified after the graph was built, and which file is the most recently modified one.
// Deleted files are not detected. Hidden, vendor and testdata directories are skipped.
func (s *Store) Stale() (newest string, stale bool, err error) {
	m, err := s.Meta()
	if err != nil || m.BuiltAt.IsZero() || m.Dir == "" {
		return "", false, err
	}
	newestTime := m.BuiltAt
	err = filepath.WalkDir(m.Dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // unreadable entries cannot make the graph stale
		}
		if d.IsDir() {
			name := d.Name()
			if path != m.Dir && (strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") && name != "go.mod" && name != "go.sum" {
			return nil
		}
		info, err := d.Info()
		if err == nil && info.ModTime().After(newestTime) {
			newestTime, newest = info.ModTime(), path
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	if rel, rerr := filepath.Rel(m.Dir, newest); rerr == nil && newest != "" {
		newest = rel
	}
	return newest, newest != "", nil
}

// UpToDate reports whether the database exists, was built with the same patterns and
// test setting, and no source file changed since. It never returns an error: any
// problem means "rebuild".
func UpToDate(path string, opts BuildOptions) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	s, err := Open(path, true)
	if err != nil {
		return false
	}
	defer s.Close()
	m, err := s.Meta()
	if err != nil || m.BuiltAt.IsZero() {
		return false
	}
	patterns := opts.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	dir := opts.Dir
	if dir == "" {
		dir = "."
	}
	if abs, err := filepath.Abs(dir); err != nil || abs != m.Dir || m.Tests != opts.Tests || !slices.Equal(m.Patterns, patterns) {
		return false
	}
	_, stale, err := s.Stale()
	return err == nil && !stale
}
