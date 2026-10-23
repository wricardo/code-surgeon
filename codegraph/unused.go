package codegraph

import (
	"strings"

	bolt "go.etcd.io/bbolt"
)

// UnusedOptions controls Unused.
type UnusedOptions struct {
	Exported bool // also report exported functions/methods (default: unexported only, since exported API may have external callers)
}

// Unused returns locally declared functions and methods that nothing uses: no static
// caller and no reference as a value. Uses by the declaration itself do not count.
//
// These are never reported: main and init, Test/Benchmark/Fuzz/Example functions in
// _test.go files, interface method declarations, and methods that may be reached through
// interface dispatch (the receiver implements an interface declaring a method with the
// same name).
//
// It is a candidate list, not proof: reflection, linkname, cgo exports, and groups of
// functions that only use each other are not detected.
func (s *Store) Unused(opts UnusedOptions) ([]Node, error) {
	var out []Node
	err := s.db.View(func(tx *bolt.Tx) error {
		var candidates []string
		for _, kind := range []string{KindFunc, KindMethod} {
			if err := scanSuffix(tx.Bucket(bByKind), kind+sep, func(id string) { candidates = append(candidates, id) }); err != nil {
				return err
			}
		}
		for _, id := range candidates {
			n, ok, err := getNode(tx, id)
			if err != nil {
				return err
			}
			if !ok || n.Ext || (!opts.Exported && n.Exported) || isEntryPoint(n) {
				continue
			}
			used, err := hasUse(tx, id)
			if err != nil {
				return err
			}
			if used {
				continue
			}
			if n.Kind == KindMethod {
				owner, ifaceMethods, err := dispatchMethods(tx, n)
				if err != nil {
					return err
				}
				if owner.Kind == KindInterface || len(ifaceMethods) > 0 {
					continue
				}
			}
			out = append(out, n)
		}
		return nil
	})
	sortNodes(out)
	return out, err
}

// hasUse reports whether id has an incoming CALLS or REFERENCES edge from another node.
func hasUse(tx *bolt.Tx, id string) (bool, error) {
	in, err := adjacent(tx, id, DirIn, "")
	if err != nil {
		return false, err
	}
	for _, e := range in {
		if (e.Rel == RelCalls || e.Rel == RelReferences) && e.From != id {
			return true, nil
		}
	}
	return false, nil
}

// isTestEntry reports Test/Benchmark/Fuzz/Example functions declared in _test.go files.
func isTestEntry(n Node) bool {
	if n.Kind != KindFunc || !strings.HasSuffix(n.File, "_test.go") {
		return false
	}
	for _, p := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
		if strings.HasPrefix(n.Short, p) {
			return true
		}
	}
	return false
}

// isEntryPoint reports functions the runtime or test framework calls.
func isEntryPoint(n Node) bool {
	if n.Kind == KindFunc && (n.Short == "main" || n.Short == "init") {
		return true
	}
	return isTestEntry(n)
}

// dispatchMethods returns the type that declares method m and the interface method
// declarations m may be called through: methods with the same name on interfaces that
// m's receiver implements. For a method declared on an interface the result is empty
// and owner.Kind is KindInterface.
func dispatchMethods(tx *bolt.Tx, m Node) (owner Node, ifaceMethods []Node, err error) {
	owners, err := adjacent(tx, m.ID, DirIn, RelHasMethod)
	if err != nil {
		return Node{}, nil, err
	}
	for _, o := range owners {
		ow, ok, err := getNode(tx, o.From)
		if err != nil {
			return Node{}, nil, err
		}
		if !ok {
			continue
		}
		owner = ow
		if ow.Kind == KindInterface {
			return owner, nil, nil
		}
		impls, err := adjacent(tx, ow.ID, DirOut, RelImplements)
		if err != nil {
			return Node{}, nil, err
		}
		for _, impl := range impls {
			ims, err := adjacent(tx, impl.To, DirOut, RelHasMethod)
			if err != nil {
				return Node{}, nil, err
			}
			for _, im := range ims {
				n, ok, err := getNode(tx, im.To)
				if err != nil {
					return Node{}, nil, err
				}
				if ok && n.Short == m.Short {
					ifaceMethods = append(ifaceMethods, n)
				}
			}
		}
	}
	return owner, ifaceMethods, nil
}
