package codegraph

import bolt "go.etcd.io/bbolt"

// Query is a chainable traversal over the graph. Each step replaces the current
// node set; the first error short-circuits the rest of the chain.
//
//	s.Find("Struct", "Calculator") -> From(ids...).Out(RelHasMethod).Where(...).Nodes()
type Query struct {
	s   *Store
	ids []string
	err error
}

// From starts a query at the given node ids.
func (s *Store) From(ids ...string) *Query {
	return &Query{s: s, ids: dedupe(ids)}
}

// FromNodes starts a query at the given nodes.
func (s *Store) FromNodes(ns []Node) *Query {
	ids := make([]string, len(ns))
	for i, n := range ns {
		ids[i] = n.ID
	}
	return s.From(ids...)
}

// Match starts a query at every node Resolve(ref) returns.
func (s *Store) Match(ref string) *Query {
	ns, err := s.Resolve(ref)
	if err != nil {
		return &Query{s: s, err: err}
	}
	return s.FromNodes(ns)
}

// Out moves one hop along outgoing edges of the given relations (any relation when none given).
func (q *Query) Out(rels ...string) *Query { return q.step(DirOut, 1, rels) }

// In moves one hop along incoming edges of the given relations (any relation when none given).
func (q *Query) In(rels ...string) *Query { return q.step(DirIn, 1, rels) }

// Transitive replaces the set with everything reachable in 1..maxDepth hops
// (maxDepth <= 0 means unlimited). Cycles are handled; start nodes appear only if reachable again.
func (q *Query) Transitive(dir Dir, maxDepth int, rels ...string) *Query {
	return q.step(dir, maxDepth, rels)
}

// Where keeps only nodes for which keep returns true.
func (q *Query) Where(keep func(Node) bool) *Query {
	if q.err != nil {
		return q
	}
	ns, err := q.Nodes()
	if err != nil {
		return &Query{s: q.s, err: err}
	}
	var ids []string
	for _, n := range ns {
		if keep(n) {
			ids = append(ids, n.ID)
		}
	}
	return &Query{s: q.s, ids: ids}
}

// IDs returns the current node ids.
func (q *Query) IDs() ([]string, error) { return q.ids, q.err }

// Nodes loads the current node set, sorted by kind, name and id.
func (q *Query) Nodes() ([]Node, error) {
	if q.err != nil {
		return nil, q.err
	}
	var out []Node
	err := q.s.db.View(func(tx *bolt.Tx) error {
		for _, id := range q.ids {
			n, ok, err := getNode(tx, id)
			if err != nil {
				return err
			}
			if ok {
				out = append(out, n)
			}
		}
		return nil
	})
	sortNodes(out)
	return out, err
}

// step is a breadth-first expansion of up to maxDepth hops; maxDepth <= 0 is unlimited.
func (q *Query) step(dir Dir, maxDepth int, rels []string) *Query {
	if q.err != nil {
		return q
	}
	var result []string
	err := q.s.db.View(func(tx *bolt.Tx) error {
		seen := map[string]struct{}{}
		frontier := q.ids
		for depth := 1; len(frontier) > 0 && (maxDepth <= 0 || depth <= maxDepth); depth++ {
			var next []string
			for _, id := range frontier {
				for _, rel := range relsOrAny(rels) {
					edges, err := adjacent(tx, id, dir, rel)
					if err != nil {
						return err
					}
					for _, e := range edges {
						other := e.To
						if dir == DirIn {
							other = e.From
						}
						if _, ok := seen[other]; !ok {
							seen[other] = struct{}{}
							next = append(next, other)
							result = append(result, other)
						}
					}
				}
			}
			frontier = next
		}
		return nil
	})
	if err != nil {
		return &Query{s: q.s, err: err}
	}
	return &Query{s: q.s, ids: result}
}

func relsOrAny(rels []string) []string {
	if len(rels) == 0 {
		return []string{""}
	}
	return rels
}

func dedupe(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := ids[:0:0]
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
