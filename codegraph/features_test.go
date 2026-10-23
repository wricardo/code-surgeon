package codegraph

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuild_ReferencesAndPackageInitialisers(t *testing.T) {
	s := buildFixture(t)

	refs, err := s.Match("useValue").Out(RelReferences).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"viaValue"}, names(refs), "function used as a value")

	calls, err := s.Match("useValue").Out(RelCalls).Nodes()
	require.NoError(t, err)
	require.Empty(t, calls, "calling a func-typed variable has no static callee")

	pkgCalls, err := s.From("pkg:example.com/p").Out(RelCalls).Nodes()
	require.NoError(t, err)
	require.Equal(t, []string{"initOnly"}, names(pkgCalls), "package-level var initialiser")
}

func TestUnused_ReportsOnlyUnreachableUnexportedByDefault(t *testing.T) {
	s := buildFixture(t)

	got, err := s.Unused(UnusedOptions{})
	require.NoError(t, err)
	// not reported: helper/a/b (called), viaValue (referenced), initOnly (var initialiser),
	// Rect.Area and Circle.Area (reachable via Shape.Area), Shape.Area (interface declaration),
	// Total (exported)
	require.ElementsMatch(t, []string{"Circle.spare", "orphan", "selfish", "useValue"}, names(got))

	all, err := s.Unused(UnusedOptions{Exported: true})
	require.NoError(t, err)
	require.Contains(t, names(all), "Total")
}

func TestImpact_WalksCallersAndInterfaceDispatch(t *testing.T) {
	s := buildFixture(t)
	helper, err := s.Resolve("helper")
	require.NoError(t, err)
	require.Len(t, helper, 1)
	require.Greater(t, helper[0].EndLine, 0)

	res, err := s.Impact([]FileLines{{File: "p.go", Ranges: [][2]int{{helper[0].Line, helper[0].Line}}}})
	require.NoError(t, err)
	require.Equal(t, []string{"helper"}, names(res.Changed))
	// helper <- Rect.Area, which may be reached through Shape.Area, called by Total
	require.ElementsMatch(t, []string{"Rect.Area", "Shape.Area", "Total", "helper"}, names(res.Affected))
	require.Empty(t, res.Tests)
}

func TestImpact_ChangedTypeAffectsMethodsAndSignatureUsers(t *testing.T) {
	s := buildFixture(t)
	rect, err := s.Resolve("Rect")
	require.NoError(t, err)
	require.Len(t, rect, 1)

	res, err := s.Impact([]FileLines{{File: "p.go", Ranges: [][2]int{{rect[0].Line, rect[0].Line}}}})
	require.NoError(t, err)
	require.Equal(t, []string{"Rect"}, names(res.Changed))
	require.Contains(t, names(res.Affected), "Rect.Area")
	require.Contains(t, names(res.Affected), "Total")
}

func TestImpact_UnrelatedFileOrLinesChangeNothing(t *testing.T) {
	s := buildFixture(t)
	res, err := s.Impact([]FileLines{{File: "other.go", Ranges: [][2]int{{1, 100}}}, {File: "p.go", Ranges: [][2]int{{1, 2}}}})
	require.NoError(t, err)
	require.Empty(t, res.Changed)
	require.Empty(t, res.Affected)
}

func TestParseDiffRanges(t *testing.T) {
	diff := `diff --git a/p.go b/p.go
--- p.go
+++ p.go
@@ -3,0 +4,2 @@ context
+x
+y
@@ -10 +12 @@
-old
+new
@@ -20,2 +21,0 @@
-gone
-gone
diff --git a/old.go b/old.go
--- old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package old
`
	require.Equal(t, []FileLines{{File: "p.go", Ranges: [][2]int{{4, 5}, {12, 12}, {21, 21}}}}, ParseDiffRanges(diff))
}

func TestStale_DetectsSourceNewerThanBuild(t *testing.T) {
	s, dir := buildFixtureDir(t)

	_, stale, err := s.Stale()
	require.NoError(t, err)
	require.False(t, stale)

	future := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "p.go"), future, future))
	newest, stale, err := s.Stale()
	require.NoError(t, err)
	require.True(t, stale)
	require.Equal(t, "p.go", newest)
}

func TestUpToDate_RequiresSameOptionsAndFreshSources(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/p\n\ngo 1.22\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "p.go"), []byte("package p\n\nfunc F() {}\n"), 0o644))
	db := filepath.Join(t.TempDir(), "g.db")
	opts := BuildOptions{Dir: dir}

	require.False(t, UpToDate(db, opts), "no database yet")

	res, err := Build(opts)
	require.NoError(t, err)
	s, err := Open(db, false)
	require.NoError(t, err)
	require.NoError(t, s.Replace(res.Nodes, res.Edges, res.Meta))
	require.NoError(t, s.Close())

	require.True(t, UpToDate(db, opts))
	require.False(t, UpToDate(db, BuildOptions{Dir: dir, Tests: true}), "different options")
	require.False(t, UpToDate(db, BuildOptions{Dir: dir, Patterns: []string{"./sub/..."}}), "different patterns")

	future := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "p.go"), future, future))
	require.False(t, UpToDate(db, opts), "source changed")
}

// With Tests enabled, package app exists as "app" and "app [app.test]". db imports the base
// variant, so its types only satisfy the base variant's interface.
func TestBuild_ImplementsAcrossTestPackageVariants(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":          "module example.com/m\n\ngo 1.22\n",
		"app/app.go":      "package app\n\ntype Item struct{ ID string }\n\ntype Repo interface{ Find(id string) Item }\n",
		"app/app_test.go": "package app\n\nimport \"testing\"\n\ntype fake struct{}\n\nfunc (fake) Find(string) Item { return Item{} }\n\nfunc TestFake(t *testing.T) { _ = fake{} }\n",
		"db/db.go":        "package db\n\nimport \"example.com/m/app\"\n\ntype repo struct{}\n\nfunc (repo) Find(id string) app.Item { return app.Item{ID: id} }\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	res, err := Build(BuildOptions{Dir: dir, Tests: true})
	require.NoError(t, err)
	require.Empty(t, res.Warnings)
	s, err := Open(filepath.Join(t.TempDir(), "g.db"), false)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	require.NoError(t, s.Replace(res.Nodes, res.Edges, res.Meta))

	impls, err := s.Match("Repo").In(RelImplements).Nodes()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"fake", "repo"}, names(impls))
}
