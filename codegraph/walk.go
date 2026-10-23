package codegraph

import (
	"errors"
	"sort"

	bolt "go.etcd.io/bbolt"
)

// ViaDispatch labels a step that crosses an interface boundary: from an interface method
// to the concrete methods that implement it, or (walking upwards) the other way round.
const ViaDispatch = "DISPATCH"

// ErrNoPath is returned by Path when the target is not reachable from the source.
var ErrNoPath = errors.New("no path found")

// hop is one traversal step to node id, reached through relation via.
type hop struct{ id, via string }

// step lists the nodes one hop away from id. With dispatch, a method is also connected
// across interface boundaries: walking out, an interface method leads to the concrete
// methods implementing it; walking in, a concrete method leads to the interface methods it
// may be called through.
func step(tx *bolt.Tx, id string, dir Dir, rels []string, dispatch bool) ([]hop, error) {
	var out []hop
	seen := map[string]struct{}{}
	add := func(id, via string) {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, hop{id, via})
		}
	}
	for _, rel := range relsOrAny(rels) {
		edges, err := adjacent(tx, id, dir, rel)
		if err != nil {
			return nil, err
		}
		for _, e := range edges {
			if dir == DirIn {
				add(e.From, e.Rel)
			} else {
				add(e.To, e.Rel)
			}
		}
	}
	if dispatch {
		n, ok, err := getNode(tx, id)
		if err != nil {
			return nil, err
		}
		if ok && n.Kind == KindMethod {
			var extra []Node
			if dir == DirOut {
				extra, err = implementationsOf(tx, n)
			} else {
				_, extra, err = dispatchMethods(tx, n)
			}
			if err != nil {
				return nil, err
			}
			for _, x := range extra {
				add(x.ID, ViaDispatch)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}

// implementationsOf returns the concrete methods that implement interface method im:
// same-named methods of the types implementing im's interface. It is empty for methods
// that are not declared on an interface.
func implementationsOf(tx *bolt.Tx, im Node) ([]Node, error) {
	owners, err := adjacent(tx, im.ID, DirIn, RelHasMethod)
	if err != nil {
		return nil, err
	}
	var out []Node
	for _, o := range owners {
		iface, ok, err := getNode(tx, o.From)
		if err != nil {
			return nil, err
		}
		if !ok || iface.Kind != KindInterface {
			continue
		}
		impls, err := adjacent(tx, iface.ID, DirIn, RelImplements)
		if err != nil {
			return nil, err
		}
		for _, impl := range impls {
			methods, err := adjacent(tx, impl.From, DirOut, RelHasMethod)
			if err != nil {
				return nil, err
			}
			for _, m := range methods {
				n, ok, err := getNode(tx, m.To)
				if err != nil {
					return nil, err
				}
				if ok && n.Short == im.Short {
					out = append(out, n)
				}
			}
		}
	}
	return out, nil
}

// useRels are the relations through which a function or method is used by another one.
var useRels = []string{RelCalls, RelReferences}

// upwardClosure returns starts plus everything that transitively uses them (CALLS and
// REFERENCES, crossing interface dispatch).
func upwardClosure(tx *bolt.Tx, starts []string) (map[string]struct{}, error) {
	seen := map[string]struct{}{}
	var queue []string
	for _, id := range starts {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			queue = append(queue, id)
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		hops, err := step(tx, id, DirIn, useRels, true)
		if err != nil {
			return nil, err
		}
		for _, h := range hops {
			if _, ok := seen[h.id]; !ok {
				seen[h.id] = struct{}{}
				queue = append(queue, h.id)
			}
		}
	}
	return seen, nil
}

// Roots returns the entry points that can reach any of ids: members of the upward
// closure that nothing uses (main, init, tests, handlers wired by generated code, unused
// code) plus package initialisers. Interface method declarations are never roots. The
// result is empty when the closure is a pure cycle.
func (s *Store) Roots(ids []string) ([]Node, error) {
	var out []Node
	err := s.db.View(func(tx *bolt.Tx) error {
		closure, err := upwardClosure(tx, ids)
		if err != nil {
			return err
		}
		for id := range closure {
			n, ok, err := getNode(tx, id)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			switch n.Kind {
			case KindPackage:
				out = append(out, n)
			case KindFunc, KindMethod:
				hops, err := step(tx, id, DirIn, useRels, true)
				if err != nil {
					return err
				}
				if len(hops) > 0 {
					continue
				}
				if n.Kind == KindMethod {
					owner, _, err := dispatchMethods(tx, n)
					if err != nil {
						return err
					}
					if owner.Kind == KindInterface {
						continue
					}
				}
				out = append(out, n)
			}
		}
		return nil
	})
	sortNodes(out)
	return out, err
}

// PathStep is one node of a path and the relation that led to it (empty for the first node).
type PathStep struct {
	Node Node   `json:"node"`
	Via  string `json:"via,omitempty"`
}

// Path finds a shortest chain of CALLS (and REFERENCES when rels includes it) from any node
// in from to any node in to. With dispatch the chain may cross interface boundaries (a call
// to an interface method continues into its implementations). It returns ErrNoPath when
// nothing connects them.
func (s *Store) Path(from, to []string, rels []string, dispatch bool) ([]PathStep, error) {
	var steps []PathStep
	err := s.db.View(func(tx *bolt.Tx) error {
		targets := map[string]struct{}{}
		for _, id := range to {
			targets[id] = struct{}{}
		}
		type parent struct{ prev, via string }
		parents := map[string]parent{}
		var queue []string
		found := ""
		for _, id := range from {
			if _, dup := parents[id]; dup {
				continue
			}
			parents[id] = parent{}
			queue = append(queue, id)
			if _, ok := targets[id]; ok && found == "" {
				found = id
			}
		}
		for found == "" && len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			hops, err := step(tx, id, DirOut, rels, dispatch)
			if err != nil {
				return err
			}
			for _, h := range hops {
				if _, seen := parents[h.id]; seen {
					continue
				}
				parents[h.id] = parent{prev: id, via: h.via}
				if _, ok := targets[h.id]; ok {
					found = h.id
					break
				}
				queue = append(queue, h.id)
			}
		}
		if found == "" {
			return ErrNoPath
		}
		for id := found; ; {
			n, ok, err := getNode(tx, id)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("graph is inconsistent: missing node " + id)
			}
			p := parents[id]
			steps = append(steps, PathStep{Node: n, Via: p.via})
			if p.prev == "" {
				break
			}
			id = p.prev
		}
		for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
			steps[i], steps[j] = steps[j], steps[i]
		}
		return nil
	})
	return steps, err
}

// TreeOptions controls Tree.
type TreeOptions struct {
	Dir          Dir      // DirOut: what the root calls; DirIn: what calls the root
	MaxDepth     int      // levels below the root; <= 0 means unlimited
	Rels         []string // relations to follow, e.g. CALLS (and REFERENCES)
	HideExternal bool     // omit stub nodes for code outside the loaded packages
	NoDispatch   bool     // do not cross interface boundaries
}

// TreeNode is a node of a call tree.
type TreeNode struct {
	Node     Node        `json:"node"`
	Via      string      `json:"via,omitempty"`      // relation that led here (empty for the root)
	Seen     bool        `json:"seen,omitempty"`     // already expanded elsewhere in the tree; children omitted
	More     int         `json:"more,omitempty"`     // children not shown because of MaxDepth
	Children []*TreeNode `json:"children,omitempty"` // sorted by name
}

// Tree expands id into a tree along opts.Rels. A node is expanded only where it first
// appears (depth first, children in id order); later occurrences are marked Seen, so the
// output stays linear in the size of the graph and cycles terminate.
func (s *Store) Tree(id string, opts TreeOptions) (*TreeNode, error) {
	var root *TreeNode
	err := s.db.View(func(tx *bolt.Tx) error {
		visited := map[string]struct{}{}
		var build func(id, via string, depth int) (*TreeNode, error)
		build = func(id, via string, depth int) (*TreeNode, error) {
			n, ok, err := getNode(tx, id)
			if err != nil || !ok {
				return nil, err
			}
			if depth > 0 && opts.HideExternal && n.Ext {
				return nil, nil
			}
			t := &TreeNode{Node: n, Via: via}
			hops, err := step(tx, id, opts.Dir, opts.Rels, !opts.NoDispatch)
			if err == nil && opts.HideExternal {
				hops, err = dropExternal(tx, hops)
			}
			if err != nil {
				return nil, err
			}
			if _, done := visited[id]; done {
				t.Seen = len(hops) > 0
				return t, nil
			}
			visited[id] = struct{}{}
			if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
				t.More = len(hops)
				return t, nil
			}
			for _, h := range hops {
				child, err := build(h.id, h.via, depth+1)
				if err != nil {
					return nil, err
				}
				if child != nil {
					t.Children = append(t.Children, child)
				}
			}
			sort.SliceStable(t.Children, func(i, j int) bool {
				a, b := t.Children[i].Node, t.Children[j].Node
				if a.Name != b.Name {
					return a.Name < b.Name
				}
				return a.ID < b.ID
			})
			return t, nil
		}
		var err error
		root, err = build(id, "", 0)
		return err
	})
	return root, err
}

// dropExternal removes hops to stub nodes (code outside the loaded packages).
func dropExternal(tx *bolt.Tx, hops []hop) ([]hop, error) {
	out := hops[:0:0]
	for _, h := range hops {
		n, ok, err := getNode(tx, h.id)
		if err != nil {
			return nil, err
		}
		if ok && !n.Ext {
			out = append(out, h)
		}
	}
	return out, nil
}
