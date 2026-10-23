package codegraph

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

// BuildOptions controls what Build loads.
type BuildOptions struct {
	Dir      string   // directory to run `go list` in (module root); "" means the current directory
	Patterns []string // package patterns; empty means "./..."
	Tests    bool     // also load _test.go files and test-only packages
}

// BuildResult is the graph extracted from the loaded packages.
type BuildResult struct {
	Nodes    []Node
	Edges    []Edge
	Meta     Meta
	Warnings []string // package load / type-check errors; the graph is still built from what type-checked
}

// Build loads the packages matching opts with full type information and
// extracts nodes and edges. Statically resolvable calls become CALLS edges
// (interface method calls point at the interface method node), and every
// concrete type that satisfies a loaded interface gets an IMPLEMENTS edge.
// Declarations outside the loaded packages become stub nodes with Ext=true.
func Build(opts BuildOptions) (*BuildResult, error) {
	dir := opts.Dir
	if dir == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	patterns := opts.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps,
		Dir:   absDir,
		Fset:  fset,
		Tests: opts.Tests,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	b := &builder{
		fset:    fset,
		dir:     absDir,
		nodes:   map[string]Node{},
		edges:   map[Edge]struct{}{},
		pending: map[string]Node{},
		ifaces:  map[string][]*types.Named{},
		concs:   map[string][]*types.Named{},
	}
	res := &BuildResult{}
	for _, p := range pkgs {
		for _, e := range p.Errors {
			res.Warnings = append(res.Warnings, e.Error())
		}
		if p.Types == nil || p.TypesInfo == nil || strings.HasSuffix(p.PkgPath, ".test") {
			continue
		}
		b.addPackage(p)
	}
	b.addImplements()
	b.addStubs()

	for _, n := range b.nodes {
		res.Nodes = append(res.Nodes, n)
	}
	sort.Slice(res.Nodes, func(i, j int) bool { return res.Nodes[i].ID < res.Nodes[j].ID })
	for e := range b.edges {
		res.Edges = append(res.Edges, e)
	}
	sort.Slice(res.Edges, func(i, j int) bool {
		a, c := res.Edges[i], res.Edges[j]
		if a.From != c.From {
			return a.From < c.From
		}
		if a.Rel != c.Rel {
			return a.Rel < c.Rel
		}
		return a.To < c.To
	})
	res.Meta = Meta{BuiltAt: time.Now().UTC(), Dir: absDir, Patterns: patterns, Tests: opts.Tests}
	return res, nil
}

type builder struct {
	fset    *token.FileSet
	dir     string
	nodes   map[string]Node
	edges   map[Edge]struct{}
	pending map[string]Node           // referenced nodes; become stubs when never declared locally
	ifaces  map[string][]*types.Named // by node id; one entry per loaded package variant
	concs   map[string][]*types.Named
}

func (b *builder) addNode(n Node) {
	if _, ok := b.nodes[n.ID]; !ok {
		b.nodes[n.ID] = n
	}
}

func (b *builder) addEdge(from, rel, to string) {
	b.edges[Edge{From: from, Rel: rel, To: to}] = struct{}{}
}

// addPackage declares every package-level type, function, method and field of p and its outgoing edges.
func (b *builder) addPackage(p *packages.Package) {
	pid := pkgID(p.PkgPath)
	b.addNode(Node{ID: pid, Kind: KindPackage, Name: p.PkgPath, Short: p.Name, Pkg: p.PkgPath})
	for path := range p.Imports {
		b.addEdge(pid, RelImports, b.refPackage(path))
	}

	docs, ends := collectDecls(p)
	qual := types.RelativeTo(p.Types)
	place := func(n *Node, pos token.Pos) {
		n.Doc = docs[pos]
		if pos.IsValid() {
			pp := b.fset.Position(pos)
			n.File, n.Line = b.relFile(pp.Filename), pp.Line
			if e, ok := ends[pos]; ok {
				n.EndLine = b.fset.Position(e).Line
			}
		}
	}

	scope := p.Types.Scope()
	for _, name := range scope.Names() {
		switch o := scope.Lookup(name).(type) {
		case *types.Func:
			n := funcNode(o)
			place(&n, o.Pos())
			n.Sig = types.ObjectString(o, qual)
			b.addNode(n)
			b.addEdge(n.ID, RelBelongsTo, pid)
			b.addSigEdges(n.ID, o)
			b.addCalls(p, n.ID, o)
		case *types.TypeName:
			named, ok := o.Type().(*types.Named)
			if !ok || o.IsAlias() {
				continue
			}
			b.addType(p, pid, named, o, place, qual)
		}
	}

	// package-level var initialisers run at init time; attribute their calls and func values to the package
	for _, f := range p.Syntax {
		for _, d := range f.Decls {
			if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.VAR {
				b.scanUses(p, pid, gd)
			}
		}
	}
}

func (b *builder) addType(p *packages.Package, pid string, named *types.Named, o *types.TypeName, place func(*Node, token.Pos), qual types.Qualifier) {
	tn := typeNode(named)
	place(&tn, o.Pos())
	b.addNode(tn)
	b.addEdge(tn.ID, RelBelongsTo, pid)
	generic := named.TypeParams().Len() > 0

	switch u := named.Underlying().(type) {
	case *types.Struct:
		if !generic {
			b.concs[tn.ID] = append(b.concs[tn.ID], named)
		}
		for i := range u.NumFields() {
			f := u.Field(i)
			fn := fieldNode(named, f)
			place(&fn, f.Pos())
			fn.Sig = types.TypeString(f.Type(), qual)
			b.addNode(fn)
			b.addEdge(tn.ID, RelHasField, fn.ID)
			b.addEdge(fn.ID, RelBelongsTo, pid)
			for _, t := range namedIn(f.Type()) {
				b.addEdge(fn.ID, RelFieldType, b.refType(t))
				if f.Embedded() {
					b.addEdge(tn.ID, RelEmbeds, b.refType(t))
				}
			}
		}
	case *types.Interface:
		if !generic && u.IsMethodSet() && u.NumMethods() > 0 {
			b.ifaces[tn.ID] = append(b.ifaces[tn.ID], named)
		}
		for i := range u.NumExplicitMethods() {
			m := u.ExplicitMethod(i)
			mn := methodNodeOn(named, m)
			place(&mn, m.Pos())
			mn.Sig = types.ObjectString(m, qual)
			b.addNode(mn)
			b.addEdge(tn.ID, RelHasMethod, mn.ID)
			b.addEdge(mn.ID, RelBelongsTo, pid)
			b.addSigEdges(mn.ID, m)
		}
		for i := range u.NumEmbeddeds() {
			for _, t := range namedIn(u.EmbeddedType(i)) {
				b.addEdge(tn.ID, RelEmbeds, b.refType(t))
			}
		}
	default:
		if !generic {
			b.concs[tn.ID] = append(b.concs[tn.ID], named)
		}
	}

	for i := range named.NumMethods() {
		m := named.Method(i)
		mn := funcNode(m)
		place(&mn, m.Pos())
		mn.Sig = types.ObjectString(m, qual)
		b.addNode(mn)
		b.addEdge(tn.ID, RelHasMethod, mn.ID)
		b.addEdge(mn.ID, RelBelongsTo, pid)
		b.addSigEdges(mn.ID, m)
		b.addCalls(p, mn.ID, m)
	}
}

// addSigEdges links fn to the named types its parameters and results mention.
func (b *builder) addSigEdges(id string, fn *types.Func) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return
	}
	for i := range sig.Params().Len() {
		for _, t := range namedIn(sig.Params().At(i).Type()) {
			b.addEdge(id, RelParamType, b.refType(t))
		}
	}
	for i := range sig.Results().Len() {
		for _, t := range namedIn(sig.Results().At(i).Type()) {
			b.addEdge(id, RelReturnsType, b.refType(t))
		}
	}
}

// addCalls records CALLS and REFERENCES edges for the body of fn.
func (b *builder) addCalls(p *packages.Package, callerID string, fn *types.Func) {
	decl := findDecl(p, fn)
	if decl == nil || decl.Body == nil {
		return
	}
	b.scanUses(p, callerID, decl.Body)
}

// scanUses records, for everything under root, CALLS for statically resolved calls and
// REFERENCES for functions/methods used as values (callbacks, method values, handler tables).
// Uses inside function literals are attributed to fromID.
func (b *builder) scanUses(p *packages.Package, fromID string, root ast.Node) {
	called := map[*ast.Ident]bool{}
	ast.Inspect(root, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if id := calleeIdent(call); id != nil {
				if fn, ok := p.TypesInfo.Uses[id].(*types.Func); ok {
					called[id] = true
					b.addEdge(fromID, RelCalls, b.refFunc(fn))
				}
			}
		}
		return true
	})
	ast.Inspect(root, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && !called[id] {
			if fn, ok := p.TypesInfo.Uses[id].(*types.Func); ok {
				b.addEdge(fromID, RelReferences, b.refFunc(fn))
			}
		}
		return true
	})
}

// addImplements links every concrete type to the loaded interfaces it satisfies. With test
// loading a package can exist in several variants ("p" and "p [p.test]") whose types are
// distinct objects, so a pair counts when any variant of the type satisfies any variant of
// the interface.
func (b *builder) addImplements() {
	for cid, variants := range b.concs {
		for iid, ivariants := range b.ifaces {
			if cid == iid || !anyImplements(variants, ivariants) {
				continue
			}
			b.addEdge(cid, RelImplements, iid)
		}
	}
}

func anyImplements(concs, ifaces []*types.Named) bool {
	for _, c := range concs {
		ptr := types.NewPointer(c)
		for _, i := range ifaces {
			it := i.Underlying().(*types.Interface)
			if types.Implements(c, it) || types.Implements(ptr, it) {
				return true
			}
		}
	}
	return false
}

// addStubs materialises referenced-but-undeclared nodes (stdlib, dependencies, `error`).
func (b *builder) addStubs() {
	for id, n := range b.pending {
		if _, ok := b.nodes[id]; !ok {
			n.Ext = true
			b.nodes[id] = n
		}
	}
}

func (b *builder) refFunc(fn *types.Func) string {
	n := funcNode(fn)
	b.pending[n.ID] = n
	return n.ID
}

func (b *builder) refType(t *types.Named) string {
	n := typeNode(t)
	b.pending[n.ID] = n
	return n.ID
}

func (b *builder) refPackage(path string) string {
	id := pkgID(path)
	b.pending[id] = Node{ID: id, Kind: KindPackage, Name: path, Short: path[strings.LastIndex(path, "/")+1:], Pkg: path}
	return id
}

func (b *builder) relFile(name string) string {
	if rel, err := filepath.Rel(b.dir, name); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return name
}

// --- node construction -------------------------------------------------------

func pkgID(path string) string { return "pkg:" + path }

func qualified(pkg *types.Package, names ...string) string {
	n := strings.Join(names, ".")
	if pkg == nil || pkg.Path() == "" {
		return n
	}
	return pkg.Path() + "." + n
}

func pkgPath(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	return pkg.Path()
}

// typeNode returns the base node for a named type (Origin of generic instances).
func typeNode(t *types.Named) Node {
	t = t.Origin()
	o := t.Obj()
	kind, prefix := KindType, "type:"
	switch t.Underlying().(type) {
	case *types.Struct:
		kind, prefix = KindStruct, "struct:"
	case *types.Interface:
		kind, prefix = KindInterface, "iface:"
	}
	return Node{ID: prefix + qualified(o.Pkg(), o.Name()), Kind: kind, Name: o.Name(), Short: o.Name(), Pkg: pkgPath(o.Pkg()), Exported: o.Exported()}
}

// recvNamed returns the named receiver type of fn, or nil for plain functions.
func recvNamed(fn *types.Func) *types.Named {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return nil
	}
	t := types.Unalias(sig.Recv().Type())
	if p, ok := t.(*types.Pointer); ok {
		t = types.Unalias(p.Elem())
	}
	n, _ := t.(*types.Named)
	return n
}

// funcNode returns the base node for a function or method (Origin of generic instances).
func funcNode(fn *types.Func) Node {
	fn = fn.Origin()
	if recv := recvNamed(fn); recv != nil {
		return methodNodeOn(recv, fn)
	}
	return Node{ID: "func:" + qualified(fn.Pkg(), fn.Name()), Kind: KindFunc, Name: fn.Name(), Short: fn.Name(), Pkg: pkgPath(fn.Pkg()), Exported: fn.Exported()}
}

func methodNodeOn(recv *types.Named, m *types.Func) Node {
	ro := recv.Origin().Obj()
	return Node{
		ID:       "method:" + qualified(ro.Pkg(), ro.Name(), m.Name()),
		Kind:     KindMethod,
		Name:     ro.Name() + "." + m.Name(),
		Short:    m.Name(),
		Pkg:      pkgPath(ro.Pkg()),
		Exported: m.Exported(),
	}
}

func fieldNode(owner *types.Named, f *types.Var) Node {
	o := owner.Obj()
	return Node{
		ID:       "field:" + qualified(o.Pkg(), o.Name(), f.Name()),
		Kind:     KindField,
		Name:     o.Name() + "." + f.Name(),
		Short:    f.Name(),
		Pkg:      pkgPath(o.Pkg()),
		Exported: f.Exported(),
	}
}

// --- AST / type helpers ------------------------------------------------------

// collectDecls maps identifier positions of funcs, types and fields to their doc comment, and
// identifier positions of funcs and types to the end position of the whole declaration.
func collectDecls(p *packages.Package) (docs map[token.Pos]string, ends map[token.Pos]token.Pos) {
	docs = map[token.Pos]string{}
	ends = map[token.Pos]token.Pos{}
	put := func(pos token.Pos, g *ast.CommentGroup) {
		if g != nil {
			docs[pos] = strings.TrimSpace(g.Text())
		}
	}
	for _, f := range p.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				put(n.Name.Pos(), n.Doc)
				ends[n.Name.Pos()] = n.End()
			case *ast.GenDecl:
				for _, s := range n.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok {
						ends[ts.Name.Pos()] = ts.End()
						if ts.Doc != nil {
							put(ts.Name.Pos(), ts.Doc)
						} else if len(n.Specs) == 1 {
							put(ts.Name.Pos(), n.Doc)
						}
					}
				}
			case *ast.Field:
				for _, name := range n.Names {
					put(name.Pos(), n.Doc)
				}
			}
			return true
		})
	}
	return docs, ends
}

// findDecl returns the FuncDecl that declares fn in p.
func findDecl(p *packages.Package, fn *types.Func) *ast.FuncDecl {
	for _, f := range p.Syntax {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Pos() == fn.Pos() {
				return fd
			}
		}
	}
	return nil
}

// calleeIdent returns the identifier naming the callee of call (function or method name), or nil
// for conversions of non-identifier expressions and calls of arbitrary expressions.
func calleeIdent(call *ast.CallExpr) *ast.Ident {
	fun := ast.Unparen(call.Fun)
	switch f := fun.(type) {
	case *ast.IndexExpr:
		fun = ast.Unparen(f.X)
	case *ast.IndexListExpr:
		fun = ast.Unparen(f.X)
	}
	switch f := fun.(type) {
	case *ast.Ident:
		return f
	case *ast.SelectorExpr:
		return f.Sel
	}
	return nil
}

// namedIn collects the named types mentioned anywhere in t (through pointers,
// slices, maps, channels, function signatures, anonymous structs and type arguments).
func namedIn(t types.Type) []*types.Named {
	var out []*types.Named
	seen := map[types.Type]bool{}
	var walk func(types.Type)
	walk = func(t types.Type) {
		switch t := t.(type) {
		case *types.Alias:
			walk(types.Unalias(t))
		case *types.Named:
			o := t.Origin()
			if seen[o] {
				return
			}
			seen[o] = true
			out = append(out, o)
			for i := range t.TypeArgs().Len() {
				walk(t.TypeArgs().At(i))
			}
		case *types.Pointer:
			walk(t.Elem())
		case *types.Slice:
			walk(t.Elem())
		case *types.Array:
			walk(t.Elem())
		case *types.Chan:
			walk(t.Elem())
		case *types.Map:
			walk(t.Key())
			walk(t.Elem())
		case *types.Signature:
			for i := range t.Params().Len() {
				walk(t.Params().At(i).Type())
			}
			for i := range t.Results().Len() {
				walk(t.Results().At(i).Type())
			}
		case *types.Struct:
			for i := range t.NumFields() {
				walk(t.Field(i).Type())
			}
		}
	}
	walk(t)
	return out
}
