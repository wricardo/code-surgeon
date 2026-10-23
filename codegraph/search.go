package codegraph

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	bolt "go.etcd.io/bbolt"
)

// SearchOptions controls Search. When none of Name, Doc and Sig is set, all three are searched.
type SearchOptions struct {
	Kind         string // restrict to one node kind; empty means all
	Name         bool   // match the name (exact, prefix or substring)
	Doc          bool   // match the doc comment
	Sig          bool   // match the signature (funcs/methods) or type expression (fields)
	HideExternal bool   // skip stub nodes for code outside the loaded packages
}

// Match kinds reported in SearchHit.Match.
const (
	MatchName = "name"
	MatchDoc  = "doc"
	MatchSig  = "sig"
)

// SearchHit is one node matching a Search. Rank orders hits: 0 exact name, 1 name prefix,
// 2 name substring, 3 doc, 4 signature.
type SearchHit struct {
	Node    Node   `json:"node"`
	Match   string `json:"match"` // MatchName, MatchDoc or MatchSig: the strongest field that matched
	Rank    int    `json:"rank"`
	Snippet string `json:"snippet,omitempty"` // matching part of the doc, or the signature
}

// Search finds nodes by concept rather than by exact name: a case-insensitive substring of
// the name, the doc comment (whitespace-normalised, so a phrase can span line breaks) or the
// signature. Each node is reported once, for its strongest match. Results are ordered by
// rank, then kind, name and id.
func (s *Store) Search(query string, opts SearchOptions) ([]SearchHit, error) {
	q := strings.ToLower(collapseSpace(query))
	if q == "" {
		return nil, errors.New("search needs a non-empty query")
	}
	if !opts.Name && !opts.Doc && !opts.Sig {
		opts.Name, opts.Doc, opts.Sig = true, true, true
	}

	var hits []SearchHit
	err := s.db.View(func(tx *bolt.Tx) error {
		visit := func(n Node) {
			if opts.HideExternal && n.Ext {
				return
			}
			if h, ok := matchNode(n, q, opts); ok {
				hits = append(hits, h)
			}
		}
		if opts.Kind != "" {
			var ids []string
			if err := scanSuffix(tx.Bucket(bByKind), opts.Kind+sep, func(id string) { ids = append(ids, id) }); err != nil {
				return err
			}
			for _, id := range ids {
				n, ok, err := getNode(tx, id)
				if err != nil {
					return err
				}
				if ok {
					visit(n)
				}
			}
			return nil
		}
		b := tx.Bucket(bNodes)
		if b == nil {
			return errors.New("graph database is empty; run `graph build` first")
		}
		return b.ForEach(func(_, raw []byte) error {
			n, err := decodeNode(raw)
			if err != nil {
				return err
			}
			visit(n)
			return nil
		})
	})
	sort.Slice(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		switch {
		case a.Rank != b.Rank:
			return a.Rank < b.Rank
		case isGeneratedFile(a.Node.File) != isGeneratedFile(b.Node.File):
			return !isGeneratedFile(a.Node.File)
		case kindPriority(a.Node.Kind) != kindPriority(b.Node.Kind):
			return kindPriority(a.Node.Kind) < kindPriority(b.Node.Kind)
		case a.Node.Name != b.Node.Name:
			return a.Node.Name < b.Node.Name
		}
		return a.Node.ID < b.Node.ID
	})
	return hits, err
}

// kindPriority orders kinds for search results: behaviour first, data fields last.
func kindPriority(kind string) int {
	switch kind {
	case KindFunc:
		return 0
	case KindMethod:
		return 1
	case KindStruct:
		return 2
	case KindInterface:
		return 3
	case KindType:
		return 4
	case KindField:
		return 5
	}
	return 6 // packages
}

// isGeneratedFile reports file names that conventionally hold generated code, so hand-written
// declarations sort first among equally strong matches.
func isGeneratedFile(file string) bool {
	f := strings.ToLower(file)
	return strings.Contains(f, "generated") || strings.HasSuffix(f, "_gen.go") ||
		strings.HasSuffix(f, "_templ.go") || strings.HasSuffix(f, ".pb.go")
}

// matchNode reports the strongest match of normalized query q in n.
func matchNode(n Node, q string, opts SearchOptions) (SearchHit, bool) {
	if opts.Name {
		name, short := strings.ToLower(n.Name), strings.ToLower(n.Short)
		switch {
		case name == q || short == q:
			return SearchHit{Node: n, Match: MatchName, Rank: 0}, true
		case strings.HasPrefix(name, q) || strings.HasPrefix(short, q):
			return SearchHit{Node: n, Match: MatchName, Rank: 1}, true
		case strings.Contains(name, q) || strings.Contains(short, q):
			return SearchHit{Node: n, Match: MatchName, Rank: 2}, true
		}
	}
	if opts.Doc && n.Doc != "" {
		doc := collapseSpace(n.Doc)
		if i := strings.Index(strings.ToLower(doc), q); i >= 0 {
			return SearchHit{Node: n, Match: MatchDoc, Rank: 3, Snippet: snippetAround(doc, i, len(q))}, true
		}
	}
	if opts.Sig && n.Sig != "" && strings.Contains(strings.ToLower(n.Sig), q) {
		return SearchHit{Node: n, Match: MatchSig, Rank: 4, Snippet: n.Sig}, true
	}
	return SearchHit{}, false
}

// collapseSpace replaces every whitespace run with one space so phrases match across line
// breaks. It keeps the case; callers lower-case when comparing.
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// snippetAround returns the part of text around the match at byte offset i of length n, cut
// on rune boundaries and marked with ellipses where text was dropped.
func snippetAround(text string, i, n int) string {
	const before, after = 40, 60
	// i comes from the lower-cased text, whose byte offsets can differ from text for exotic
	// Unicode; clamp so a misaligned offset can only shift the snippet, never panic.
	i = min(max(i, 0), len(text))
	start, end := max(i-before, 0), min(i+n+after, len(text))
	for start > 0 && start < len(text) && !utf8.RuneStart(text[start]) {
		start--
	}
	for end < len(text) && !utf8.RuneStart(text[end]) {
		end++
	}
	out := text[start:end]
	if start > 0 {
		out = "…" + out
	}
	if end < len(text) {
		out += "…"
	}
	return out
}
