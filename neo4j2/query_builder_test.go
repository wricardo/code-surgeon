package neo4j2

import (
	"context"
	"strings"
	"testing"

	codesurgeon "github.com/wricardo/code-surgeon"

	"github.com/stretchr/testify/require"
)

// --- BuildMergeQuery tests ---

func TestBuildMergeQuery(t *testing.T) {
	t.Run("basic merge", func(t *testing.T) {
		mq := MergeQuery{
			NodeType: "Package",
			Alias:    "p",
			Properties: map[string]any{
				"name": "mypackage",
			},
		}

		qf := BuildMergeQuery(mq)
		require.Contains(t, qf.Fragment, "MERGE (p:Package")
		require.Contains(t, qf.Fragment, "name: $pname")
		require.Equal(t, "mypackage", qf.Params["pname"])
	})

	t.Run("merge with set fields", func(t *testing.T) {
		mq := MergeQuery{
			NodeType: "Struct",
			Alias:    "s",
			Properties: map[string]any{
				"name": "MyStruct",
			},
			SetFields: map[string]string{
				"documentation": "some docs",
				"isExported":    "true",
			},
		}

		qf := BuildMergeQuery(mq)
		require.Contains(t, qf.Fragment, "MERGE (s:Struct")
		require.Contains(t, qf.Fragment, "SET")
		require.Contains(t, qf.Fragment, "s.documentation = $sdocumentation")
		require.Contains(t, qf.Fragment, "s.isExported = $sisExported")
		require.Equal(t, "MyStruct", qf.Params["sname"])
		require.Equal(t, "some docs", qf.Params["sdocumentation"])
		require.Equal(t, "true", qf.Params["sisExported"])
	})

	t.Run("default alias", func(t *testing.T) {
		mq := MergeQuery{
			NodeType: "Node",
			Properties: map[string]any{
				"id": "123",
			},
		}

		qf := BuildMergeQuery(mq)
		require.Contains(t, qf.Fragment, "MERGE (n:Node")
		require.Equal(t, "123", qf.Params["nid"])
	})

	t.Run("multiple properties", func(t *testing.T) {
		mq := MergeQuery{
			NodeType: "Function",
			Alias:    "f",
			Properties: map[string]any{
				"name":        "MyFunc",
				"packageName": "pkg",
			},
		}

		qf := BuildMergeQuery(mq)
		require.Contains(t, qf.Fragment, "MERGE (f:Function")
		require.Equal(t, "MyFunc", qf.Params["fname"])
		require.Equal(t, "pkg", qf.Params["fpackageName"])
	})
}

// --- BuildMatchQuery tests ---

func TestBuildMatchQuery(t *testing.T) {
	t.Run("basic match", func(t *testing.T) {
		mq := MatchQuery{
			NodeType: "Package",
			Alias:    "p",
			Properties: map[string]any{
				"name": "mypackage",
			},
		}

		qf := BuildMatchQuery(mq)
		require.Contains(t, qf.Fragment, "MATCH (p:Package")
		require.Contains(t, qf.Fragment, "name: $pname")
		require.Equal(t, "mypackage", qf.Params["pname"])
		require.NotContains(t, qf.Fragment, "OPTIONAL")
	})

	t.Run("default alias", func(t *testing.T) {
		mq := MatchQuery{
			NodeType: "Node",
			Properties: map[string]any{
				"id": "test",
			},
		}

		qf := BuildMatchQuery(mq)
		require.Contains(t, qf.Fragment, "MATCH (n:Node")
	})
}

// --- BuildOptionalMatchQuery tests ---

func TestBuildOptionalMatchQuery(t *testing.T) {
	t.Run("basic optional match", func(t *testing.T) {
		mq := MatchQuery{
			NodeType: "Method",
			Alias:    "m",
			Properties: map[string]any{
				"name":     "MyMethod",
				"receiver": "MyStruct",
			},
		}

		qf := BuildOptionalMatchQuery(mq)
		require.Contains(t, qf.Fragment, "OPTIONAL MATCH (m:Method")
		require.Contains(t, qf.Fragment, "name: $mname")
		require.Contains(t, qf.Fragment, "receiver: $mreceiver")
		require.Equal(t, "MyMethod", qf.Params["mname"])
		require.Equal(t, "MyStruct", qf.Params["mreceiver"])
	})
}

// --- CypherQuery builder chain tests ---

func TestCypherQueryBuilder(t *testing.T) {
	t.Run("merge and return", func(t *testing.T) {
		q := CypherQuery{}.
			Merge(MergeQuery{
				NodeType: "Package",
				Alias:    "p",
				Properties: map[string]any{
					"name": "test",
				},
			}).
			Return("id(p) as nodeID")

		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "MERGE (p:Package")
		require.Contains(t, query, "RETURN id(p) as nodeID")
		require.Equal(t, "test", q.Args["pname"])
	})

	t.Run("merge with relationship", func(t *testing.T) {
		q := CypherQuery{}.
			Merge(MergeQuery{
				NodeType:   "Function",
				Alias:      "f",
				Properties: map[string]any{"name": "Foo"},
			}).
			Merge(MergeQuery{
				NodeType:   "Package",
				Alias:      "p",
				Properties: map[string]any{"name": "bar"},
			}).
			MergeRel("f", "BELONGS_TO", "p", nil).
			Return("id(f) as nodeID")

		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "MERGE (f)-[:BELONGS_TO]->(p)")
		require.Contains(t, query, "RETURN id(f) as nodeID")
	})

	t.Run("merge rel with properties", func(t *testing.T) {
		q := CypherQuery{}.
			Merge(MergeQuery{
				NodeType:   "A",
				Alias:      "a",
				Properties: map[string]any{"id": "1"},
			}).
			Merge(MergeQuery{
				NodeType:   "B",
				Alias:      "b",
				Properties: map[string]any{"id": "2"},
			}).
			MergeRel("a", "REL", "b", map[string]interface{}{
				"weight": 5,
			}).
			Return()

		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "MERGE (a)-[:REL {weight: $a_REL_weight}]->(b)")
		require.Equal(t, 5, q.Args["a_REL_weight"])
		require.Contains(t, query, "RETURN 1")
	})

	t.Run("with clause", func(t *testing.T) {
		q := CypherQuery{}.
			Merge(MergeQuery{
				NodeType:   "Node",
				Alias:      "n",
				Properties: map[string]any{"id": "1"},
			}).
			With("n").
			Return("n.id as id")

		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "WITH n")
	})

	t.Run("with multiple aliases", func(t *testing.T) {
		q := CypherQuery{}.With("a", "b", "c")
		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "WITH a, b, c")
	})

	t.Run("where clause", func(t *testing.T) {
		q := CypherQuery{}.Where("n.name = 'test'", "n.id > 0")
		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "WHERE n.name = 'test' AND n.id > 0")
	})

	t.Run("raw query", func(t *testing.T) {
		q := CypherQuery{}.Raw("CALL db.labels()")
		query := strings.Join(q.query, "\n")
		require.Equal(t, "CALL db.labels()", query)
	})

	t.Run("unwind", func(t *testing.T) {
		q := CypherQuery{}.Unwind("items", "item")
		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "UNWIND $items as item")
	})

	t.Run("match and optional match", func(t *testing.T) {
		q := CypherQuery{}.
			Match(MatchQuery{
				NodeType:   "Struct",
				Alias:      "s",
				Properties: map[string]any{"name": "Foo"},
			}).
			OptionalMatch(MatchQuery{
				NodeType:   "Method",
				Alias:      "m",
				Properties: map[string]any{"receiver": "Foo"},
			}).
			Return("s", "m")

		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "MATCH (s:Struct")
		require.Contains(t, query, "OPTIONAL MATCH (m:Method")
		require.Contains(t, query, "RETURN s, m")
	})

	t.Run("return with no fields", func(t *testing.T) {
		q := CypherQuery{}.Return()
		query := strings.Join(q.query, "\n")
		require.Equal(t, "RETURN 1", query)
	})

	t.Run("complex chained query", func(t *testing.T) {
		q := CypherQuery{}.
			Merge(MergeQuery{
				NodeType:   "Function",
				Alias:      "f",
				Properties: map[string]any{"name": "Foo"},
				SetFields:  map[string]string{"documentation": "docs"},
			}).
			Merge(MergeQuery{
				NodeType:   "Package",
				Alias:      "p",
				Properties: map[string]any{"name": "pkg"},
			}).
			MergeRel("f", "BELONGS_TO", "p", nil).
			With("f", "p").
			Match(MatchQuery{
				NodeType:   "Type",
				Alias:      "t",
				Properties: map[string]any{"type": "int"},
			}).
			MergeRel("f", "RETURNS", "t", nil).
			Return("id(f) as id")

		query := strings.Join(q.query, "\n")
		require.Contains(t, query, "MERGE (f:Function")
		require.Contains(t, query, "MERGE (p:Package")
		require.Contains(t, query, "MERGE (f)-[:BELONGS_TO]->(p)")
		require.Contains(t, query, "WITH f, p")
		require.Contains(t, query, "MATCH (t:Type")
		require.Contains(t, query, "MERGE (f)-[:RETURNS]->(t)")
		require.Contains(t, query, "RETURN id(f) as id")
	})
}

// --- ReplacePointerNotation tests ---

func TestReplacePointerNotation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(*Person)", "Person"},
		{"(*conn)", "conn"},
		{"(*MyStruct)", "MyStruct"},
		{"Person", "Person"},
		{"*Person", "*Person"},
		{"(*A).Method", "A.Method"},
		{"", ""},
		{"(*a)(*b)", "ab"},
		{"(*Type123)", "Type123"},
		{"(*under_score)", "under_score"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.expected, ReplacePointerNotation(tt.input))
		})
	}
}

// --- smartSplit tests ---

func TestSmartSplit(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a.b.c", []string{"a", "b", "c"}},
		{"(*Foo).Bar", []string{"(*Foo)", "Bar"}},
		{"Handler[...].func1", []string{"Handler[...]", "func1"}},
		{"simple", []string{"simple"}},
		{"", nil},
		{"a(b.c).d", []string{"a(b.c)", "d"}},
		{"x[y.z].w", []string{"x[y.z]", "w"}},
		{"a.b(c.d).e.f", []string{"a", "b(c.d)", "e", "f"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := smartSplit(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// --- ParseFolder tests ---

func TestParseFolder(t *testing.T) {
	tests := []struct {
		input          string
		expectedFolder string
		expectedName   string
	}{
		{"/path/to/file.go", "/path/to", "to"},
		{"/a/b/c/d.go", "/a/b/c", "c"},
		{"file.go", "", ""},
		{"/root/file.go", "/root", "root"},
		{"a/b", "a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			folder, name := ParseFolder(tt.input)
			require.Equal(t, tt.expectedFolder, folder)
			require.Equal(t, tt.expectedName, name)
		})
	}
}

// --- ParseRepository tests ---

func TestParseRepository(t *testing.T) {
	tests := []struct {
		pkg          string
		folder       string
		expectedRepo string
		expectedOrg  string
		expectedName string
	}{
		{"github.com/user/repo/pkg", "", "github.com/user/repo", "user", "repo"},
		{"bitbucket.org/org/repo/sub", "", "bitbucket.org/org/repo", "org", "repo"},
		{"gitlab.com/org/repo", "", "gitlab.com/org/repo", "org", "repo"},
		{"gorm.io/gorm/clause", "", "gorm.io/gorm", "gorm.io", "gorm"},
		{"gorm.io/gorm", "", "gorm.io/gorm", "gorm.io", "gorm"},
		{"net/http", "", "net/http", "net/http", "http"},
		{"main", "", "", "", ""},
		{"", "", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			repo, org, name := ParseRepository(tt.pkg, tt.folder)
			require.Equal(t, tt.expectedRepo, repo, "repo mismatch")
			require.Equal(t, tt.expectedOrg, org, "org mismatch")
			require.Equal(t, tt.expectedName, name, "name mismatch")
		})
	}
}

// --- isBuiltinType tests ---

func TestIsBuiltinType(t *testing.T) {
	builtins := []string{
		"bool", "byte", "complex64", "complex128", "error",
		"float32", "float64", "int", "int8", "int16", "int32", "int64",
		"rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
	}

	for _, b := range builtins {
		t.Run(b, func(t *testing.T) {
			require.True(t, isBuiltinType(b), "%s should be builtin", b)
		})
	}

	nonBuiltins := []string{
		"MyStruct", "Person", "interface{}", "", "map", "chan", "func",
	}

	for _, nb := range nonBuiltins {
		t.Run(nb, func(t *testing.T) {
			require.False(t, isBuiltinType(nb), "%s should not be builtin", nb)
		})
	}
}

// --- Merge* helper function tests ---

func TestMergePackage(t *testing.T) {
	ctx := context.Background()
	mod := codesurgeon.Module{
		RootModuleName: "github.com/user/repo",
		FullName:       "github.com/user/repo/pkg",
	}
	pkg := codesurgeon.Package{
		Package: "pkg",
	}

	mq := MergePackage(ctx, "p", mod, pkg)
	require.Equal(t, "Package", mq.NodeType)
	require.Equal(t, "p", mq.Alias)
	require.Equal(t, "github.com/user/repo", mq.Properties["rootPackageFullName"])
	require.Equal(t, "github.com/user/repo/pkg", mq.Properties["module_full_name"])
	require.Equal(t, "pkg", mq.Properties["name"])
}

func TestMergeStruct(t *testing.T) {
	ctx := context.Background()
	mod := codesurgeon.Module{
		RootModuleName: "github.com/user/repo",
		FullName:       "github.com/user/repo/pkg",
	}
	pkg := codesurgeon.Package{Package: "pkg"}
	strct := codesurgeon.Struct{
		Name:       "MyStruct",
		Docs:       []string{"MyStruct docs"},
		Definition: "type MyStruct struct{}",
		IsExported: true,
	}

	mq := MergeStruct(ctx, "s", mod, pkg, strct)
	require.Equal(t, "Struct", mq.NodeType)
	require.Equal(t, "s", mq.Alias)
	require.Equal(t, "MyStruct", mq.Properties["name"])
	require.Equal(t, "pkg", mq.Properties["packageName"])
	require.Equal(t, "MyStruct docs", mq.SetFields["documentation"])
	require.Equal(t, "type MyStruct struct{}", mq.SetFields["definition"])
	require.Equal(t, "true", mq.SetFields["isExported"])
}

func TestMergeModule(t *testing.T) {
	ctx := context.Background()
	mod := codesurgeon.Module{
		RootModuleName: "github.com/user/repo",
	}

	mq := MergeModule(ctx, "m", mod)
	require.Equal(t, "Module", mq.NodeType)
	require.Equal(t, "m", mq.Alias)
	require.Equal(t, "github.com/user/repo", mq.Properties["name"])
}

func TestMergeFunction(t *testing.T) {
	ctx := context.Background()
	mod := codesurgeon.Module{
		RootModuleName: "github.com/user/repo",
		FullName:       "github.com/user/repo/pkg",
	}
	pkg := codesurgeon.Package{Package: "pkg"}
	fn := codesurgeon.Function{
		Name:        "MyFunc",
		Docs:        []string{"MyFunc does things"},
		Definition:  "func MyFunc() error",
		IsExported:  true,
		IsTest:      false,
		IsBenchmark: false,
	}

	mq := MergeFunction(ctx, "f", mod, pkg, fn)
	require.Equal(t, "Function", mq.NodeType)
	require.Equal(t, "f", mq.Alias)
	require.Equal(t, "MyFunc", mq.Properties["name"])
	require.Equal(t, "pkg", mq.Properties["packageName"])
	require.Equal(t, "github.com/user/repo/pkg", mq.Properties["packageFullName"])
	require.Equal(t, "MyFunc does things", mq.SetFields["documentation"])
	require.Equal(t, "false", mq.SetFields["isTest"])
}

func TestMergeMethod(t *testing.T) {
	ctx := context.Background()
	mod := codesurgeon.Module{
		RootModuleName: "github.com/user/repo",
		FullName:       "github.com/user/repo/pkg",
	}
	pkg := codesurgeon.Package{Package: "pkg"}
	method := codesurgeon.Method{
		Name:       "DoWork",
		Receiver:   "*Worker",
		Docs:       []string{"DoWork processes"},
		Definition: "func (*Worker) DoWork()",
		IsExported: true,
		IsTest:     false,
	}

	mq := MergeMethod(ctx, "m", mod, pkg, method)
	require.Equal(t, "Method", mq.NodeType)
	require.Equal(t, "m", mq.Alias)
	require.Equal(t, "DoWork", mq.Properties["name"])
	require.Equal(t, "Worker", mq.Properties["receiver"]) // asterisk stripped
	require.Equal(t, "DoWork processes", mq.SetFields["documentation"])
}

func TestMergeInterface(t *testing.T) {
	ctx := context.Background()
	mod := codesurgeon.Module{
		RootModuleName: "github.com/user/repo",
		FullName:       "github.com/user/repo/pkg",
	}
	pkg := codesurgeon.Package{Package: "pkg"}
	iface := codesurgeon.Interface{
		Name:       "Reader",
		Docs:       []string{"Reader reads"},
		Definition: "type Reader interface { Read() }",
		IsExported: true,
	}

	mq := MergeInterface(ctx, "i", mod, pkg, iface)
	require.Equal(t, "Interface", mq.NodeType)
	require.Equal(t, "i", mq.Alias)
	require.Equal(t, "Reader", mq.Properties["name"])
	require.Equal(t, "true", mq.SetFields["isExported"])
}

func TestMergeField(t *testing.T) {
	ctx := context.Background()
	mod := codesurgeon.Module{FullName: "github.com/user/repo/pkg"}
	pkg := codesurgeon.Package{Package: "pkg"}
	strct := codesurgeon.Struct{Name: "MyStruct"}
	field := codesurgeon.Field{
		Name:    "Name",
		Type:    "string",
		Docs:    []string{"Name field"},
		Private: false,
	}

	mq := MergeField(ctx, "f", mod, pkg, strct, field)
	require.Equal(t, "Field", mq.NodeType)
	require.Equal(t, "f", mq.Alias)
	require.Equal(t, "Name", mq.Properties["name"])
	require.Equal(t, "MyStruct", mq.Properties["structName"])
	require.Equal(t, "string", mq.SetFields["type"])
	require.Equal(t, "true", mq.SetFields["isExported"])
}

func TestMergeBaseType(t *testing.T) {
	ctx := context.Background()

	t.Run("builtin type", func(t *testing.T) {
		td := codesurgeon.TypeDetails{
			Type:     strPtr("string"),
			TypeName: "string",
		}
		mq := MergeBaseType(ctx, "b", "string", "pkg", "github.com/user/repo/pkg", td)
		require.Equal(t, "BaseType", mq.NodeType)
		require.Equal(t, "string", mq.Properties["type"])
		require.Equal(t, "builtin", mq.Properties["packageName"])
		require.Equal(t, "builtin", mq.Properties["packageFullName"])
	})

	t.Run("external type", func(t *testing.T) {
		extPkg := "github.com/other/lib"
		extPkgName := "lib"
		td := codesurgeon.TypeDetails{
			Type:        strPtr("Widget"),
			TypeName:    "lib.Widget",
			Package:     &extPkg,
			PackageName: &extPkgName,
			IsExternal:  true,
			TypeReferences: []codesurgeon.TypeReference{
				{Package: &extPkg, PackageName: &extPkgName, Name: "Widget"},
			},
		}
		mq := MergeBaseType(ctx, "b", "lib.Widget", "pkg", "github.com/user/repo/pkg", td)
		require.Equal(t, "Widget", mq.Properties["type"])
		require.Equal(t, "lib", mq.Properties["packageName"])
		require.Equal(t, "github.com/other/lib", mq.Properties["packageFullName"])
	})

	t.Run("local type", func(t *testing.T) {
		pkgName := "pkg"
		td := codesurgeon.TypeDetails{
			Type:        strPtr("MyType"),
			TypeName:    "MyType",
			PackageName: &pkgName,
		}
		mq := MergeBaseType(ctx, "b", "MyType", "pkg", "github.com/user/repo/pkg", td)
		require.Equal(t, "MyType", mq.Properties["type"])
		require.Equal(t, "pkg", mq.Properties["packageName"])
		require.Equal(t, "github.com/user/repo/pkg", mq.Properties["packageFullName"])
	})

	t.Run("fallback parsing from type string", func(t *testing.T) {
		td := codesurgeon.TypeDetails{}
		mq := MergeBaseType(ctx, "b", "*model.Person", "pkg", "github.com/user/repo/pkg", td)
		require.Equal(t, "Person", mq.Properties["type"])
		require.Equal(t, "model", mq.Properties["packageName"])
	})
}

func TestMergeType(t *testing.T) {
	ctx := context.Background()

	t.Run("with type name", func(t *testing.T) {
		td := codesurgeon.TypeDetails{
			TypeName: "string",
		}
		mq := MergeType(ctx, "t", "string", td)
		require.Equal(t, "Type", mq.NodeType)
		require.Equal(t, "string", mq.Properties["type"])
	})

	t.Run("fallback to Type field", func(t *testing.T) {
		td := codesurgeon.TypeDetails{
			Type: strPtr("int"),
		}
		mq := MergeType(ctx, "t", "int", td)
		require.Equal(t, "int", mq.Properties["type"])
	})

	t.Run("fallback to typeString", func(t *testing.T) {
		td := codesurgeon.TypeDetails{}
		mq := MergeType(ctx, "t", "float64", td)
		require.Equal(t, "float64", mq.Properties["type"])
	})
}

func TestMergeCall(t *testing.T) {
	ctx := context.Background()
	pe := ParsedStackEntry{
		Function:    "DoWork",
		Receiver:    "Worker",
		Package:     "github.com/user/repo/pkg",
		PackageName: "pkg",
	}

	mq := MergeCall(ctx, "mc", "stack123", pe)
	require.Equal(t, "Call", mq.NodeType)
	require.Equal(t, "mc", mq.Alias)
	require.Equal(t, "DoWork", mq.Properties["function"])
	require.Equal(t, "Worker", mq.Properties["receiver"])
	require.Equal(t, "github.com/user/repo/pkg", mq.Properties["packageFullName"])
	require.Equal(t, "stack123", mq.Properties["stack_id"])
	require.NotEmpty(t, mq.SetFields["id"])
}

// --- parseStackTrace tests ---

func TestParseStackTrace(t *testing.T) {
	stack := `goroutine 1 [running]:
main.functionC({0x104486073?, 0xc?})
	/Users/user/project/main.go:24 +0x9f
main.(*Person).SayHello(0x1400010aeb8)
	/Users/user/project/main.go:19 +0x2c
main.main()
	/Users/user/project/main.go:10 +0x28`

	parsed := parseStackTrace(stack)
	require.Len(t, parsed, 3)

	// Note: parseStackTrace has a variable naming quirk where ParseReceiver's
	// second return (receiver) is assigned to the variable used for Function field,
	// and the first return (function name) goes to OriginalName.
	// For simple functions (no receiver), Function is empty and OriginalName has the name.

	// First entry: simple function
	require.Equal(t, "main", parsed[0].Package)
	require.Equal(t, "main", parsed[0].PackageName)
	require.Equal(t, "24", parsed[0].Line)
	require.Contains(t, parsed[0].File, "main.go")
	require.Equal(t, "functionC", parsed[0].OriginalName)

	// Second entry: method with pointer receiver
	require.Equal(t, "main", parsed[1].Package)
	require.Equal(t, "19", parsed[1].Line)
	require.Equal(t, "SayHello", parsed[1].OriginalName)

	// Third entry
	require.Equal(t, "main", parsed[2].Package)
	require.Equal(t, "10", parsed[2].Line)
}

// --- getStackHash tests ---

func TestGetStackHash(t *testing.T) {
	entries := []ParsedStackEntry{
		{Package: "main", Function: "foo", Line: "10"},
		{Package: "main", Function: "bar", Line: "20"},
	}

	hash1 := getStackHash(entries)
	require.NotEmpty(t, hash1)
	require.Len(t, hash1, 32) // MD5 hex is 32 chars

	// Same input should give same hash
	hash2 := getStackHash(entries)
	require.Equal(t, hash1, hash2)

	// Different input should give different hash
	entries2 := []ParsedStackEntry{
		{Package: "main", Function: "baz", Line: "30"},
	}
	hash3 := getStackHash(entries2)
	require.NotEqual(t, hash1, hash3)
}

// --- toString tests ---

func TestToString(t *testing.T) {
	require.Equal(t, "", toString(nil))
	require.Equal(t, "hello", toString("hello"))
	require.Equal(t, "", toString(42))
	require.Equal(t, "", toString(true))
}

// --- normalizeType tests ---

func TestNormalizeType(t *testing.T) {
	// Currently normalizeType is identity function
	require.Equal(t, "string", normalizeType("string"))
	require.Equal(t, "*int", normalizeType("*int"))
	require.Equal(t, "", normalizeType(""))
}

// --- QueryFragment tests ---

func TestQueryFragment(t *testing.T) {
	qf := QueryFragment{
		Fragment: "MERGE (n:Node {id: $nid})",
		Params:   map[string]interface{}{"nid": "123"},
	}

	require.Contains(t, qf.Fragment, "MERGE")
	require.Equal(t, "123", qf.Params["nid"])
}

// helper
func strPtr(s string) *string {
	return &s
}
