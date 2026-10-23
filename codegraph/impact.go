package codegraph

import (
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// FileLines lists changed line ranges (inclusive, 1-based, in the new version of the file).
type FileLines struct {
	File   string   `json:"file"` // relative to the graph build directory
	Ranges [][2]int `json:"ranges"`
}

// ParseDiffRanges extracts changed new-file line ranges from `git diff -U0 --no-prefix` output.
// A pure deletion is reported as the single line where it happened. Deleted files are skipped.
func ParseDiffRanges(diff string) []FileLines {
	var out []FileLines
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if path == "/dev/null" {
				out = append(out, FileLines{}) // placeholder so following hunks are dropped
			} else {
				out = append(out, FileLines{File: path})
			}
		case strings.HasPrefix(line, "@@ ") && len(out) > 0 && out[len(out)-1].File != "":
			// @@ -a,b +c,d @@
			fields := strings.Fields(line)
			if len(fields) < 3 || !strings.HasPrefix(fields[2], "+") {
				continue
			}
			start, count := 0, 1
			spec := strings.SplitN(strings.TrimPrefix(fields[2], "+"), ",", 2)
			start, _ = strconv.Atoi(spec[0])
			if len(spec) == 2 {
				count, _ = strconv.Atoi(spec[1])
			}
			end := start + count - 1
			if count == 0 {
				end = start
			}
			if start < 1 {
				start = 1
			}
			f := &out[len(out)-1]
			f.Ranges = append(f.Ranges, [2]int{start, max(end, start)})
		}
	}
	var res []FileLines
	for _, f := range out {
		if f.File != "" && len(f.Ranges) > 0 {
			res = append(res, f)
		}
	}
	return res
}

// ImpactResult is what Impact found for a set of changes.
type ImpactResult struct {
	Changed  []Node `json:"changed"`  // declarations whose line range overlaps a change
	Affected []Node `json:"affected"` // changed funcs/methods plus everything that transitively uses them
	Tests    []Node `json:"tests"`    // affected Test/Benchmark/Fuzz/Example functions (needs a graph built with --tests)
}

// Impact maps changed lines to declarations and walks upwards through CALLS and
// REFERENCES edges to every function, method and package initialiser that may be affected.
// A changed concrete method also affects callers of interface methods it may be
// dispatched through. A changed type affects its methods and the funcs/methods whose
// parameters or results mention it.
func (s *Store) Impact(changes []FileLines) (*ImpactResult, error) {
	res := &ImpactResult{}
	err := s.db.View(func(tx *bolt.Tx) error {
		byFile := map[string][][2]int{}
		for _, c := range changes {
			byFile[c.File] = append(byFile[c.File], c.Ranges...)
		}

		var changed []Node
		for _, kind := range []string{KindFunc, KindMethod, KindStruct, KindInterface, KindType} {
			var ids []string
			if err := scanSuffix(tx.Bucket(bByKind), kind+sep, func(id string) { ids = append(ids, id) }); err != nil {
				return err
			}
			for _, id := range ids {
				n, ok, err := getNode(tx, id)
				if err != nil {
					return err
				}
				if ok && !n.Ext && overlaps(n, byFile[n.File]) {
					changed = append(changed, n)
				}
			}
		}
		sortNodes(changed)
		res.Changed = changed

		var starts []string
		for _, n := range changed {
			switch n.Kind {
			case KindFunc, KindMethod:
				starts = append(starts, n.ID)
			default: // a changed type affects its methods and the signatures that mention it
				out, err := adjacent(tx, n.ID, DirOut, RelHasMethod)
				if err != nil {
					return err
				}
				for _, e := range out {
					starts = append(starts, e.To)
				}
				for _, rel := range []string{RelParamType, RelReturnsType} {
					in, err := adjacent(tx, n.ID, DirIn, rel)
					if err != nil {
						return err
					}
					for _, e := range in {
						starts = append(starts, e.From)
					}
				}
			}
		}
		affected, err := upwardClosure(tx, starts)
		if err != nil {
			return err
		}

		for id := range affected {
			n, ok, err := getNode(tx, id)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			res.Affected = append(res.Affected, n)
			if isTestEntry(n) {
				res.Tests = append(res.Tests, n)
			}
		}
		return nil
	})
	sortNodes(res.Affected)
	sortNodes(res.Tests)
	return res, err
}

// overlaps reports whether the declaration lines of n intersect any of the ranges.
func overlaps(n Node, ranges [][2]int) bool {
	end := max(n.EndLine, n.Line)
	for _, r := range ranges {
		if n.Line <= r[1] && end >= r[0] {
			return true
		}
	}
	return false
}
