package codegraph

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// idOf resolves ref to exactly one node id.
func idOf(t *testing.T, s *Store, ref string) string {
	t.Helper()
	ns, err := s.Resolve(ref)
	require.NoError(t, err)
	require.Len(t, ns, 1, ref)
	return ns[0].ID
}

func childNames(n *TreeNode) []string {
	out := make([]string, len(n.Children))
	for i, c := range n.Children {
		out[i] = c.Node.Name
	}
	return out
}

func child(t *testing.T, n *TreeNode, name string) *TreeNode {
	t.Helper()
	for _, c := range n.Children {
		if c.Node.Name == name {
			return c
		}
	}
	require.Failf(t, "child not found", "%s not in %v", name, childNames(n))
	return nil
}

func TestTree_CrossesInterfaceDispatchAndLabelsEdges(t *testing.T) {
	s := buildFixture(t)
	tree, err := s.Tree(idOf(t, s, "Total"), TreeOptions{Dir: DirOut, Rels: []string{RelCalls}})
	require.NoError(t, err)
	require.Equal(t, "Total", tree.Node.Name)
	require.ElementsMatch(t, []string{"Shape.Area", "New"}, childNames(tree))

	shapeArea := child(t, tree, "Shape.Area")
	require.Equal(t, RelCalls, shapeArea.Via)
	require.ElementsMatch(t, []string{"Rect.Area", "Circle.Area"}, childNames(shapeArea))
	rectArea := child(t, shapeArea, "Rect.Area")
	require.Equal(t, ViaDispatch, rectArea.Via)
	require.Equal(t, []string{"helper"}, childNames(rectArea))
}

func TestTree_OptionsHideExternalDisableDispatchAndLimitDepth(t *testing.T) {
	s := buildFixture(t)
	total := idOf(t, s, "Total")

	local, err := s.Tree(total, TreeOptions{Dir: DirOut, Rels: []string{RelCalls}, HideExternal: true})
	require.NoError(t, err)
	require.Equal(t, []string{"Shape.Area"}, childNames(local))

	noDispatch, err := s.Tree(total, TreeOptions{Dir: DirOut, Rels: []string{RelCalls}, NoDispatch: true})
	require.NoError(t, err)
	require.Empty(t, child(t, noDispatch, "Shape.Area").Children)

	shallow, err := s.Tree(total, TreeOptions{Dir: DirOut, Rels: []string{RelCalls}, MaxDepth: 1})
	require.NoError(t, err)
	require.Equal(t, 2, child(t, shallow, "Shape.Area").More, "two implementations are hidden by the depth limit")
	require.Empty(t, child(t, shallow, "Shape.Area").Children)
}

func TestTree_CycleIsMarkedSeenAndTerminates(t *testing.T) {
	s := buildFixture(t)
	tree, err := s.Tree(idOf(t, s, "a"), TreeOptions{Dir: DirOut, Rels: []string{RelCalls}})
	require.NoError(t, err)
	b := child(t, tree, "b")
	back := child(t, b, "a")
	require.True(t, back.Seen)
	require.Empty(t, back.Children)
}

func TestTree_CallersDirectionWalksUpThroughDispatch(t *testing.T) {
	s := buildFixture(t)
	tree, err := s.Tree(idOf(t, s, "helper"), TreeOptions{Dir: DirIn, Rels: []string{RelCalls}})
	require.NoError(t, err)
	rectArea := child(t, tree, "Rect.Area")
	shapeArea := child(t, rectArea, "Shape.Area")
	require.Equal(t, ViaDispatch, shapeArea.Via)
	require.Equal(t, []string{"Total"}, childNames(shapeArea))
}

func TestRoots_FindsEntryPointsThroughDispatchAndIgnoresInterfaceDeclarations(t *testing.T) {
	s := buildFixture(t)

	roots, err := s.Roots([]string{idOf(t, s, "helper")})
	require.NoError(t, err)
	require.Equal(t, []string{"Total"}, names(roots), "Rect.Area is reached via Shape.Area, which Total calls")

	self, err := s.Roots([]string{idOf(t, s, "Total")})
	require.NoError(t, err)
	require.Equal(t, []string{"Total"}, names(self), "a function nothing uses is its own entry point")

	cycle, err := s.Roots([]string{idOf(t, s, "a")})
	require.NoError(t, err)
	require.Empty(t, cycle, "a pure cycle has no entry point")

	pkgInit, err := s.Roots([]string{idOf(t, s, "initOnly")})
	require.NoError(t, err)
	require.Equal(t, []string{"example.com/p"}, names(pkgInit), "called from a package-level var initialiser")
}

func TestPath_ShortestChainWithDispatchLabels(t *testing.T) {
	s := buildFixture(t)
	from, to := idOf(t, s, "Total"), idOf(t, s, "helper")

	steps, err := s.Path([]string{from}, []string{to}, []string{RelCalls}, true)
	require.NoError(t, err)
	var got, vias []string
	for _, st := range steps {
		got = append(got, st.Node.Name)
		vias = append(vias, st.Via)
	}
	require.Equal(t, []string{"Total", "Shape.Area", "Rect.Area", "helper"}, got)
	require.Equal(t, []string{"", RelCalls, ViaDispatch, RelCalls}, vias)

	_, err = s.Path([]string{from}, []string{to}, []string{RelCalls}, false)
	require.ErrorIs(t, err, ErrNoPath, "without dispatch the interface call is a dead end")

	_, err = s.Path([]string{to}, []string{from}, []string{RelCalls}, true)
	require.ErrorIs(t, err, ErrNoPath, "paths are directed")

	same, err := s.Path([]string{from}, []string{from}, []string{RelCalls}, true)
	require.NoError(t, err)
	require.Len(t, same, 1)
}

func TestUsages_GroupsParamsReturnsFieldsAndEmbedding(t *testing.T) {
	s := buildModule(t, map[string]string{
		"go.mod": "module example.com/u\n\ngo 1.22\n",
		"u.go": `package u

type Item struct{}

type Box struct {
	It   Item
	Many []Item
}

type Wrapper struct{ Item }

func Take(i Item)  {}
func Make() Item   { return Item{} }
func (b Box) Method(x *Item) {}
`,
	})
	u, err := s.Usages("struct:example.com/u.Item") // "Item" alone is ambiguous: the embedded field Wrapper.Item has the same short name
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Take", "Box.Method"}, names(u.Params))
	require.ElementsMatch(t, []string{"Make"}, names(u.Returns))
	require.ElementsMatch(t, []string{"Box.It", "Box.Many", "Wrapper.Item"}, names(u.Fields))
	require.ElementsMatch(t, []string{"Wrapper"}, names(u.EmbeddedBy))
}

func TestDescribe_GroupsNeighboursByRelationAndDirection(t *testing.T) {
	s := buildFixture(t)
	d, err := s.Describe(idOf(t, s, "Rect"))
	require.NoError(t, err)
	require.Equal(t, "Rect", d.Node.Name)
	require.Equal(t, []string{"Rect.Area"}, names(d.Out[RelHasMethod]))
	require.Equal(t, []string{"Rect.H", "Rect.W"}, names(d.Out[RelHasField]))
	require.Equal(t, []string{"Shape"}, names(d.Out[RelImplements]))
	require.Equal(t, []string{"example.com/p"}, names(d.Out[RelBelongsTo]))

	helper, err := s.Describe(idOf(t, s, "helper"))
	require.NoError(t, err)
	require.Equal(t, []string{"Rect.Area"}, names(helper.In[RelCalls]))

	_, err = s.Describe("func:nope.Nothing")
	require.ErrorContains(t, err, "no node with id")
}
