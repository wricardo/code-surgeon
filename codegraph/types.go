package codegraph

import (
	"fmt"
	"strings"
)

// DeclsOptions controls Decls.
type DeclsOptions struct {
	Recursive    bool     // also include sub-packages of the matched package(s)
	Kinds        []string // KindStruct, KindInterface, KindType, KindFunc, KindMethod; empty means all five
	ExportedOnly bool
}

// TypesOptions controls Types.
type TypesOptions struct {
	Recursive    bool     // also include sub-packages of the matched package(s)
	Kinds        []string // KindStruct, KindInterface, KindType; empty means all three
	ExportedOnly bool
}

var (
	typeKinds = []string{KindStruct, KindInterface, KindType}
	declKinds = []string{KindStruct, KindInterface, KindType, KindFunc, KindMethod}
)

// Types lists the named types declared in the package identified by ref. See Decls.
func (s *Store) Types(ref string, opts TypesOptions) ([]Node, error) {
	kinds := opts.Kinds
	if len(kinds) == 0 {
		kinds = typeKinds
	}
	return s.Decls(ref, DeclsOptions{Recursive: opts.Recursive, Kinds: kinds, ExportedOnly: opts.ExportedOnly})
}

// Decls lists declarations (types, functions, methods) of the package identified by ref:
// its full import path or its short package name (a short name shared by several packages
// matches all of them). Methods include interface method declarations. Struct fields are
// not listed; use Describe on the struct. Packages outside the loaded set have no
// declarations in the graph. It returns an error when no loaded package matches.
func (s *Store) Decls(ref string, opts DeclsOptions) ([]Node, error) {
	all, err := s.Find(KindPackage, "")
	if err != nil {
		return nil, err
	}
	var roots []Node
	for _, p := range all {
		if !p.Ext && (p.Name == ref || p.Short == ref) {
			roots = append(roots, p)
		}
	}

	pkgs := roots
	if opts.Recursive {
		// A directory without Go files has no package node, so an import path also matches
		// by prefix; a short name expands from the packages it matched.
		var prefixes []string
		if strings.Contains(ref, "/") {
			prefixes = append(prefixes, ref)
		}
		for _, r := range roots {
			prefixes = append(prefixes, r.Name)
		}
		pkgs = nil
		for _, p := range all {
			if p.Ext {
				continue
			}
			for _, prefix := range prefixes {
				if p.Name == prefix || strings.HasPrefix(p.Name, prefix+"/") {
					pkgs = append(pkgs, p)
					break
				}
			}
		}
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no loaded package matches %q (try `graph find -k Package <prefix>`)", ref)
	}

	kinds := opts.Kinds
	if len(kinds) == 0 {
		kinds = declKinds
	}
	return s.FromNodes(pkgs).In(RelBelongsTo).Where(func(n Node) bool {
		if opts.ExportedOnly && !n.Exported {
			return false
		}
		for _, k := range kinds {
			if n.Kind == k {
				return true
			}
		}
		return false
	}).Nodes()
}
