package codegraph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Bucket layout. Keys use 0x00 as separator, which never occurs in node ids.
//
//	nodes  : id                      -> JSON Node
//	out    : from \0 rel \0 to       -> empty
//	in     : to   \0 rel \0 from     -> empty
//	byname : name \0 id              -> empty   (Name and Short are both indexed)
//	bykind : kind \0 id              -> empty
//	meta   : "info"                  -> JSON Meta
var (
	bNodes  = []byte("nodes")
	bOut    = []byte("out")
	bIn     = []byte("in")
	bByName = []byte("byname")
	bByKind = []byte("bykind")
	bMeta   = []byte("meta")
	keyInfo = []byte("info")
)

const sep = "\x00"

// Meta describes the last build stored in a database.
type Meta struct {
	BuiltAt  time.Time `json:"built_at"`
	Dir      string    `json:"dir"`
	Patterns []string  `json:"patterns"`
	Tests    bool      `json:"tests"`
}

// Stats summarises the contents of a database.
type Stats struct {
	Meta  Meta           `json:"meta"`
	Nodes map[string]int `json:"nodes"` // by kind
	Edges map[string]int `json:"edges"` // by relation
}

// Store is a bbolt-backed code graph. Writers hold an exclusive file lock;
// open read-only for queries.
type Store struct {
	db *bolt.DB
}

// Open opens (and, for read-write, creates) the database at path.
func Open(path string, readOnly bool) (*Store, error) {
	if !readOnly {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	db, err := bolt.Open(path, 0o644, &bolt.Options{ReadOnly: readOnly, Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open graph db %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

// Close releases the database file lock.
func (s *Store) Close() error { return s.db.Close() }

// Replace atomically swaps the whole graph. Readers never observe a partial build.
func (s *Store) Replace(nodes []Node, edges []Edge, meta Meta) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{bNodes, bOut, bIn, bByName, bByKind, bMeta} {
			if tx.Bucket(name) != nil {
				if err := tx.DeleteBucket(name); err != nil {
					return err
				}
			}
			if _, err := tx.CreateBucket(name); err != nil {
				return err
			}
		}
		nb, ob, ib, nameB, kindB := tx.Bucket(bNodes), tx.Bucket(bOut), tx.Bucket(bIn), tx.Bucket(bByName), tx.Bucket(bByKind)
		empty := []byte{}
		for _, n := range nodes {
			raw, err := json.Marshal(n)
			if err != nil {
				return err
			}
			if err := nb.Put([]byte(n.ID), raw); err != nil {
				return err
			}
			if err := nameB.Put([]byte(n.Name+sep+n.ID), empty); err != nil {
				return err
			}
			if n.Short != "" && n.Short != n.Name {
				if err := nameB.Put([]byte(n.Short+sep+n.ID), empty); err != nil {
					return err
				}
			}
			if err := kindB.Put([]byte(n.Kind+sep+n.ID), empty); err != nil {
				return err
			}
		}
		for _, e := range edges {
			if err := ob.Put([]byte(e.From+sep+e.Rel+sep+e.To), empty); err != nil {
				return err
			}
			if err := ib.Put([]byte(e.To+sep+e.Rel+sep+e.From), empty); err != nil {
				return err
			}
		}
		raw, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		return tx.Bucket(bMeta).Put(keyInfo, raw)
	})
}

// Node returns the node with the given id; ok is false when it does not exist.
func (s *Store) Node(id string) (n Node, ok bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		n, ok, err = getNode(tx, id)
		return err
	})
	return n, ok, err
}

func getNode(tx *bolt.Tx, id string) (Node, bool, error) {
	b := tx.Bucket(bNodes)
	if b == nil {
		return Node{}, false, errors.New("graph database is empty; run `graph build` first")
	}
	raw := b.Get([]byte(id))
	if raw == nil {
		return Node{}, false, nil
	}
	n, err := decodeNode(raw)
	if err != nil {
		return Node{}, false, err
	}
	return n, true, nil
}

func decodeNode(raw []byte) (Node, error) {
	var n Node
	err := json.Unmarshal(raw, &n)
	return n, err
}

// Find returns nodes whose Name or Short starts with namePrefix, optionally
// restricted to kind. An empty namePrefix with a kind lists every node of that kind.
// Results are sorted by kind, name, id.
func (s *Store) Find(kind, namePrefix string) ([]Node, error) {
	var out []Node
	err := s.db.View(func(tx *bolt.Tx) error {
		ids := map[string]struct{}{}
		if namePrefix == "" {
			if kind == "" {
				return errors.New("find needs a kind or a name prefix")
			}
			if err := scanSuffix(tx.Bucket(bByKind), kind+sep, func(id string) { ids[id] = struct{}{} }); err != nil {
				return err
			}
		} else {
			b := tx.Bucket(bByName)
			if b == nil {
				return errors.New("graph database is empty; run `graph build` first")
			}
			c := b.Cursor()
			p := []byte(namePrefix)
			for k, _ := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, _ = c.Next() {
				i := bytes.IndexByte(k, 0)
				ids[string(k[i+1:])] = struct{}{}
			}
		}
		for id := range ids {
			n, ok, err := getNode(tx, id)
			if err != nil {
				return err
			}
			if ok && (kind == "" || n.Kind == kind) {
				out = append(out, n)
			}
		}
		return nil
	})
	sortNodes(out)
	return out, err
}

// scanSuffix calls fn with the key remainder after prefix for every key starting with prefix.
func scanSuffix(b *bolt.Bucket, prefix string, fn func(string)) error {
	if b == nil {
		return errors.New("graph database is empty; run `graph build` first")
	}
	c := b.Cursor()
	p := []byte(prefix)
	for k, _ := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, _ = c.Next() {
		fn(string(k[len(p):]))
	}
	return nil
}

// Resolve turns a user supplied reference into nodes: an exact id, otherwise every
// node whose Name or Short equals ref exactly (a method may be given as "Type.Method" or "Method").
func (s *Store) Resolve(ref string) ([]Node, error) {
	if n, ok, err := s.Node(ref); err != nil || ok {
		if err != nil {
			return nil, err
		}
		return []Node{n}, nil
	}
	cands, err := s.Find("", ref)
	if err != nil {
		return nil, err
	}
	var out []Node
	for _, n := range cands {
		if n.Name == ref || n.Short == ref {
			out = append(out, n)
		}
	}
	return out, nil
}

// Adjacent returns the edges incident to id in the given direction, optionally
// restricted to one relation. Edges keep their real orientation (From -> To).
func (s *Store) Adjacent(id string, dir Dir, rel string) ([]Edge, error) {
	var out []Edge
	err := s.db.View(func(tx *bolt.Tx) error {
		var err error
		out, err = adjacent(tx, id, dir, rel)
		return err
	})
	return out, err
}

func adjacent(tx *bolt.Tx, id string, dir Dir, rel string) ([]Edge, error) {
	name := bOut
	if dir == DirIn {
		name = bIn
	}
	b := tx.Bucket(name)
	if b == nil {
		return nil, errors.New("graph database is empty; run `graph build` first")
	}
	prefix := id + sep
	if rel != "" {
		prefix += rel + sep
	}
	var out []Edge
	c := b.Cursor()
	p := []byte(prefix)
	for k, _ := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, _ = c.Next() {
		parts := strings.SplitN(string(k), sep, 3)
		if len(parts) != 3 {
			continue
		}
		e := Edge{From: parts[0], Rel: parts[1], To: parts[2]}
		if dir == DirIn {
			e = Edge{From: parts[2], Rel: parts[1], To: parts[0]}
		}
		out = append(out, e)
	}
	return out, nil
}

// Stats counts nodes per kind and edges per relation.
func (s *Store) Stats() (Stats, error) {
	st := Stats{Nodes: map[string]int{}, Edges: map[string]int{}}
	err := s.db.View(func(tx *bolt.Tx) error {
		mb, kb, ob := tx.Bucket(bMeta), tx.Bucket(bByKind), tx.Bucket(bOut)
		if mb == nil || kb == nil || ob == nil {
			return errors.New("graph database is empty; run `graph build` first")
		}
		if raw := mb.Get(keyInfo); raw != nil {
			if err := json.Unmarshal(raw, &st.Meta); err != nil {
				return err
			}
		}
		c := kb.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			st.Nodes[string(k[:bytes.IndexByte(k, 0)])]++
		}
		c = ob.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			parts := strings.SplitN(string(k), sep, 3)
			if len(parts) == 3 {
				st.Edges[parts[1]]++
			}
		}
		return nil
	})
	return st, err
}

func sortNodes(ns []Node) {
	sort.Slice(ns, func(i, j int) bool {
		a, b := ns[i], ns[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
}
