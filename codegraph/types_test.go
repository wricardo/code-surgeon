package codegraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// buildModule writes files (path -> content) into a temp module and returns its stored graph.
func buildModule(t *testing.T, files map[string]string) *Store {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	res, err := Build(BuildOptions{Dir: dir})
	require.NoError(t, err)
	require.Empty(t, res.Warnings)
	s, err := Open(filepath.Join(t.TempDir(), "g.db"), false)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	require.NoError(t, s.Replace(res.Nodes, res.Edges, res.Meta))
	return s
}

var typesModule = map[string]string{
	"go.mod":   "module example.com/m\n\ngo 1.22\n",
	"a/a.go":   "package a\n\ntype Server struct{ Name string }\n\nfunc (s *Server) Start() {}\n\ntype handler struct{}\n\nfunc (handler) serve() {}\n\ntype Runner interface{ Run() }\n\ntype ID string\n\nfunc New() *Server { return nil }\n\nfunc helper() {}\n",
	"a/b/b.go": "package b\n\ntype Deep struct{}\n\nfunc Dig() {}\n",
	"ab/ab.go": "package ab\n\ntype Other struct{}\n",
}

func TestTypes_ListsStructsInterfacesAndNamedTypesOfOnePackage(t *testing.T) {
	s := buildModule(t, typesModule)

	got, err := s.Types("example.com/m/a", TypesOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Server", "handler", "Runner", "ID"}, names(got), "functions, methods and fields are not types")

	byShort, err := s.Types("a", TypesOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, names(got), names(byShort), "short package name resolves to the same package")
}

func TestTypes_FiltersByKindAndExportedness(t *testing.T) {
	s := buildModule(t, typesModule)

	structs, err := s.Types("a", TypesOptions{Kinds: []string{KindStruct}})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Server", "handler"}, names(structs))

	exported, err := s.Types("a", TypesOptions{ExportedOnly: true})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Server", "Runner", "ID"}, names(exported))

	ifaces, err := s.Types("a", TypesOptions{Kinds: []string{KindInterface, KindType}})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Runner", "ID"}, names(ifaces))
}

func TestTypes_RecursiveIncludesSubpackagesOnPathBoundaryOnly(t *testing.T) {
	s := buildModule(t, typesModule)

	flat, err := s.Types("example.com/m/a", TypesOptions{})
	require.NoError(t, err)
	require.NotContains(t, names(flat), "Deep")

	rec, err := s.Types("example.com/m/a", TypesOptions{Recursive: true})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Server", "handler", "Runner", "ID", "Deep"}, names(rec), "a/b is included, ab is not")
}

func TestTypes_UnknownPackageIsAnError(t *testing.T) {
	s := buildModule(t, typesModule)
	_, err := s.Types("example.com/m/nope", TypesOptions{})
	require.ErrorContains(t, err, "no loaded package matches")

	// a stub for a package that was only imported is not a loaded package
	_, err = s.Types("fmt", TypesOptions{})
	require.Error(t, err)
}

func TestDecls_ListsFunctionsAndMethodsWithSignatures(t *testing.T) {
	s := buildModule(t, typesModule)

	all, err := s.Decls("example.com/m/a", DeclsOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"Server", "handler", "Runner", "ID", // types
		"New", "helper", // functions
		"Server.Start", "handler.serve", "Runner.Run", // methods, including the interface method declaration
	}, names(all), "fields are not declarations")

	funcs, err := s.Decls("a", DeclsOptions{Kinds: []string{KindFunc}})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"New", "helper"}, names(funcs))
	for _, f := range funcs {
		if f.Name == "New" {
			require.Contains(t, f.Sig, "func New() *Server")
		}
	}

	methods, err := s.Decls("a", DeclsOptions{Kinds: []string{KindMethod}})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Server.Start", "handler.serve", "Runner.Run"}, names(methods))

	exported, err := s.Decls("a", DeclsOptions{Kinds: []string{KindFunc, KindMethod}, ExportedOnly: true})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"New", "Server.Start", "Runner.Run"}, names(exported))
}

func TestDecls_RecursiveAddsSubpackageFunctions(t *testing.T) {
	s := buildModule(t, typesModule)

	flat, err := s.Decls("example.com/m/a", DeclsOptions{Kinds: []string{KindFunc}})
	require.NoError(t, err)
	require.NotContains(t, names(flat), "Dig")

	rec, err := s.Decls("example.com/m/a", DeclsOptions{Kinds: []string{KindFunc}, Recursive: true})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"New", "helper", "Dig"}, names(rec))
}

func TestDecls_RecursiveFromDirectoryWithoutGoFilesMatchesByImportPathPrefix(t *testing.T) {
	s := buildModule(t, typesModule) // the module root example.com/m has no .go files, so no package node

	rec, err := s.Decls("example.com/m", DeclsOptions{Kinds: []string{KindStruct}, Recursive: true})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Server", "handler", "Deep", "Other"}, names(rec))

	_, err = s.Decls("example.com/m", DeclsOptions{Kinds: []string{KindStruct}})
	require.ErrorContains(t, err, "no loaded package matches", "without -r the directory is not a package")
}
