package codegraph

import (
	bolt "go.etcd.io/bbolt"
)

// Usages groups where a type is used.
type Usages struct {
	Params     []Node `json:"params"`      // funcs/methods with a parameter mentioning the type
	Returns    []Node `json:"returns"`     // funcs/methods with a result mentioning the type
	Fields     []Node `json:"fields"`      // fields whose type mentions the type
	EmbeddedBy []Node `json:"embedded_by"` // types embedding it
}

// Usages reports where the type with the given id is used. "Mentions" includes pointers,
// slices, maps, channels and type arguments around it.
func (s *Store) Usages(id string) (*Usages, error) {
	u := &Usages{}
	err := s.db.View(func(tx *bolt.Tx) error {
		for _, g := range []struct {
			rel string
			dst *[]Node
		}{{RelParamType, &u.Params}, {RelReturnsType, &u.Returns}, {RelFieldType, &u.Fields}, {RelEmbeds, &u.EmbeddedBy}} {
			edges, err := adjacent(tx, id, DirIn, g.rel)
			if err != nil {
				return err
			}
			ns, err := loadNodes(tx, edgeSources(edges))
			if err != nil {
				return err
			}
			*g.dst = ns
		}
		return nil
	})
	return u, err
}

// Description is everything the graph stores about one node: the node and its edges
// grouped by relation, with the neighbours loaded.
type Description struct {
	Node Node              `json:"node"`
	Out  map[string][]Node `json:"out,omitempty"` // relation -> targets
	In   map[string][]Node `json:"in,omitempty"`  // relation -> sources
}

// Describe loads the node with the given id and its neighbours.
func (s *Store) Describe(id string) (*Description, error) {
	d := &Description{Out: map[string][]Node{}, In: map[string][]Node{}}
	err := s.db.View(func(tx *bolt.Tx) error {
		n, ok, err := getNode(tx, id)
		if err != nil {
			return err
		}
		if !ok {
			return errNoNode(id)
		}
		d.Node = n
		for _, dir := range []Dir{DirOut, DirIn} {
			edges, err := adjacent(tx, id, dir, "")
			if err != nil {
				return err
			}
			byRel := map[string][]string{}
			for _, e := range edges {
				other := e.To
				if dir == DirIn {
					other = e.From
				}
				byRel[e.Rel] = append(byRel[e.Rel], other)
			}
			dst := d.Out
			if dir == DirIn {
				dst = d.In
			}
			for rel, ids := range byRel {
				ns, err := loadNodes(tx, ids)
				if err != nil {
					return err
				}
				dst[rel] = ns
			}
		}
		return nil
	})
	return d, err
}

type errNoNode string

func (e errNoNode) Error() string { return "no node with id " + string(e) }

func edgeSources(edges []Edge) []string {
	ids := make([]string, len(edges))
	for i, e := range edges {
		ids[i] = e.From
	}
	return ids
}

// loadNodes loads the nodes for ids (skipping unknown ids), sorted by kind, name and id.
func loadNodes(tx *bolt.Tx, ids []string) ([]Node, error) {
	out := make([]Node, 0, len(ids))
	for _, id := range ids {
		n, ok, err := getNode(tx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, n)
		}
	}
	sortNodes(out)
	return out, nil
}
