package codegraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const fixture = `package p

import "errors"

// Shape has an area.
type Shape interface{ Area() float64 }

type Rect struct{ W, H float64 }

func (r Rect) Area() float64 { return helper(r.W, r.H) }

func helper(a, b float64) float64 { return a * b }

type Circle struct{ R float64 }

func (c *Circle) Area() float64 { return 3 * c.R * c.R }

func Total(s []Shape) (float64, error) {
	t := 0.0
	for _, x := range s {
		t += x.Area()
	}
	if t < 0 {
		return 0, errors.New("negative")
	}
	return t, nil
}

func a() { b() }
func b() { a() }

func orphan() {}
func selfish() { selfish() }
func (c *Circle) spare() {}

func viaValue() {}
func useValue() { f := viaValue; f() }

func initOnly() int { return 1 }

var cfg = initOnly()
`

// buildFixture writes the fixture module, builds it and stores the graph in a fresh database.
func buildFixture(t *testing.T) *Store {
	s, _ := buildFixtureDir(t)
	return s
}

// buildFixtureDir is buildFixture that also returns the module directory.
func buildFixtureDir(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/p\n\ngo 1.22\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p.go"), []byte(fixture), 0o644))

	res, err := Build(BuildOptions{Dir: dir})
	require.NoError(t, err)
	require.Empty(t, res.Warnings)

	s, err := Open(filepath.Join(t.TempDir(), "g.db"), false)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	require.NoError(t, s.Replace(res.Nodes, res.Edges, res.Meta))
	return s, dir
}

func names(ns []Node) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = n.Name
	}
	return out
}

func TestBuild_StructureAndPositions(t *testing.T) {
	s := buildFixture(t)

	rect, err := s.Find(KindStruct, "Rect")
	require.NoError(t, err)
	require.Len(t, rect, 1)
	require.Equal(t, "p.go", rect[0].File)

	methods, err := s.From(rect[0].ID).Out(RelHasMethod).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"Rect.Area"}, names(methods))
	require.Contains(t, methods[0].Sig, "Area")

	fields, err := s.From(rect[0].ID).Out(RelHasField).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"Rect.H", "Rect.W"}, names(fields))

	shape, err := s.Resolve("Shape")
	require.NoError(t, err)
	require.Len(t, shape, 1)
	require.Equal(t, "Shape has an area.", shape[0].Doc)
}

func TestBuild_ResolveMethodByShortAndQualifiedName(t *testing.T) {
	s := buildFixture(t)
	for _, ref := range []string{"Area", "Rect.Area"} {
		ns, err := s.Resolve(ref)
		require.NoError(t, err)
		require.NotEmpty(t, ns, ref)
	}
	ns, err := s.Resolve("Circle.Area")
	require.NoError(t, err)
	require.Len(t, ns, 1)
}

func TestBuild_ImplementsIncludesPointerReceivers(t *testing.T) {
	s := buildFixture(t)
	impls, err := s.Match("Shape").In(RelImplements).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"Circle", "Rect"}, names(impls))

	ifaces, err := s.Match("Circle").Out(RelImplements).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"Shape"}, names(ifaces))
}

func TestBuild_CallsStaticAndInterfaceDispatch(t *testing.T) {
	s := buildFixture(t)

	callers, err := s.Match("helper").In(RelCalls).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"Rect.Area"}, names(callers))

	// x.Area() on a Shape resolves to the interface method, not to a concrete implementation.
	callees, err := s.Match("Total").Out(RelCalls).Where(func(n Node) bool { return n.Short == "Area" }).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"Shape.Area"}, names(callees))

	// calls to other packages produce external stub nodes
	ext, err := s.Match("Total").Out(RelCalls).Where(func(n Node) bool { return n.Ext }).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"New"}, names(ext))
	require.Equal(t, "errors", ext[0].Pkg)
}

func TestQuery_TransitiveTerminatesOnCyclesAndHonoursDepth(t *testing.T) {
	s := buildFixture(t)

	all, err := s.Match("a").In(RelCalls).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"b"}, names(all))

	closure, err := s.Match("a").Transitive(DirIn, 0, RelCalls).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, names(closure), "a is reachable from itself through b")

	one, err := s.Match("a").Transitive(DirOut, 1, RelCalls).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"b"}, names(one))
}

func TestBuild_TypeEdgesIncludeUniverseError(t *testing.T) {
	s := buildFixture(t)
	ret, err := s.Match("Total").Out(RelReturnsType).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"error"}, names(ret))

	param, err := s.Match("Total").Out(RelParamType).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"Shape"}, names(param), "element type of []Shape")
}

func TestStore_ReplaceDropsPreviousGraph(t *testing.T) {
	s := buildFixture(t)
	require.NoError(t, s.Replace([]Node{{ID: "func:x.Only", Kind: KindFunc, Name: "Only", Short: "Only"}}, nil, Meta{}))

	old, err := s.Find("", "Rect")
	require.NoError(t, err)
	require.Empty(t, old)

	st, err := s.Stats()
	require.NoError(t, err)
	require.Equal(t, map[string]int{KindFunc: 1}, st.Nodes)
}

func TestStore_ReadOnlyOpenOfMissingDatabaseFails(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "missing.db"), true)
	require.Error(t, err)
}
