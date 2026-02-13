package codesurgeon

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- ToSnakeCase tests ---

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HelloWorld", "hello_world"},
		{"helloWorld", "hello_world"},
		{"Hello", "hello"},
		{"hello", "hello"},
		{"HTTPServer", "h_t_t_p_server"},
		{"ID", "i_d"},
		{"", ""},
		{"A", "a"},
		{"already_snake", "already_snake"},
		{"ABC", "a_b_c"},
		{"MyHTTPHandler", "my_h_t_t_p_handler"},
		{"simpleTest", "simple_test"},
		{"JSONParser", "j_s_o_n_parser"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// --- RenderTemplate tests ---

func TestRenderTemplate(t *testing.T) {
	t.Run("simple template", func(t *testing.T) {
		result, err := RenderTemplate("Hello {{.Name}}", map[string]string{"Name": "World"})
		require.NoError(t, err)
		require.Equal(t, "Hello World", result)
	})

	t.Run("template with no data", func(t *testing.T) {
		result, err := RenderTemplate("Hello World", nil)
		require.NoError(t, err)
		require.Equal(t, "Hello World", result)
	})

	t.Run("invalid template syntax", func(t *testing.T) {
		_, err := RenderTemplate("Hello {{.Name", nil)
		require.Error(t, err)
	})

	t.Run("missing field", func(t *testing.T) {
		_, err := RenderTemplate("Hello {{.Name}}", struct{}{})
		require.Error(t, err)
	})

	t.Run("struct data", func(t *testing.T) {
		data := struct {
			First string
			Last  string
		}{"John", "Doe"}
		result, err := RenderTemplate("{{.First}} {{.Last}}", data)
		require.NoError(t, err)
		require.Equal(t, "John Doe", result)
	})

	t.Run("empty template", func(t *testing.T) {
		result, err := RenderTemplate("", nil)
		require.NoError(t, err)
		require.Equal(t, "", result)
	})
}

func TestMustRenderTemplate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		result := MustRenderTemplate("Hello {{.Name}}", map[string]string{"Name": "World"})
		require.Equal(t, "Hello World", result)
	})

	t.Run("panics on invalid template", func(t *testing.T) {
		require.Panics(t, func() {
			MustRenderTemplate("{{.Name", nil)
		})
	})

	t.Run("panics on execution error", func(t *testing.T) {
		require.Panics(t, func() {
			MustRenderTemplate("{{.Name}}", struct{}{})
		})
	})
}

func TestRenderTemplateNoError(t *testing.T) {
	t.Run("valid template returns result", func(t *testing.T) {
		result := RenderTemplateNoError("Hello {{.Name}}", map[string]string{"Name": "World"})
		require.Equal(t, "Hello World", result)
	})

	t.Run("invalid template returns empty", func(t *testing.T) {
		result := RenderTemplateNoError("{{.Name", nil)
		require.Equal(t, "", result)
	})
}

// --- EnsureGoFileExists tests ---

func TestEnsureGoFileExists(t *testing.T) {
	t.Run("creates new file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "test.go")

		err := EnsureGoFileExists(filePath, "mypackage")
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "package mypackage\n", string(content))
	})

	t.Run("does not overwrite existing file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "existing.go")

		err := os.WriteFile(filePath, []byte("package existing\n\nvar x = 1\n"), 0644)
		require.NoError(t, err)

		err = EnsureGoFileExists(filePath, "other")
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Equal(t, "package existing\n\nvar x = 1\n", string(content))
	})
}

// --- ApplyFileChanges tests ---

func TestApplyFileChanges(t *testing.T) {
	t.Run("creates file and inserts code", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "generated.go")

		changes := []FileChange{
			{
				PackageName: "generated",
				File:        filePath,
				Fragments: []CodeFragment{
					{
						Content:   "func Hello() string { return \"hello\" }",
						Overwrite: false,
					},
				},
			},
		}

		err := ApplyFileChanges(changes)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "package generated")
		require.Contains(t, string(content), "func Hello()")
	})

	t.Run("creates nested directory", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "sub", "dir", "file.go")

		changes := []FileChange{
			{
				PackageName: "dir",
				File:        filePath,
				Fragments: []CodeFragment{
					{
						Content:   "func Nested() {}",
						Overwrite: false,
					},
				},
			},
		}

		err := ApplyFileChanges(changes)
		require.NoError(t, err)

		_, err = os.Stat(filePath)
		require.NoError(t, err)
	})

	t.Run("overwrite false preserves existing function", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "existing.go")

		err := os.WriteFile(filePath, []byte("package existing\n\nfunc Hello() string { return \"original\" }\n"), 0644)
		require.NoError(t, err)

		changes := []FileChange{
			{
				PackageName: "existing",
				File:        filePath,
				Fragments: []CodeFragment{
					{
						Content:   "func Hello() string { return \"replaced\" }",
						Overwrite: false,
					},
				},
			},
		}

		err = ApplyFileChanges(changes)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "original")
		require.NotContains(t, string(content), "replaced")
	})

	t.Run("overwrite true replaces existing function", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "existing.go")

		err := os.WriteFile(filePath, []byte("package existing\n\nfunc Hello() string { return \"original\" }\n"), 0644)
		require.NoError(t, err)

		changes := []FileChange{
			{
				PackageName: "existing",
				File:        filePath,
				Fragments: []CodeFragment{
					{
						Content:   "func Hello() string { return \"replaced\" }",
						Overwrite: true,
					},
				},
			},
		}

		err = ApplyFileChanges(changes)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "replaced")
		require.NotContains(t, string(content), "original")
	})
}

// --- InsertCodeFragments tests ---

func TestInsertCodeFragments(t *testing.T) {
	t.Run("adds function to existing file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "test.go")

		err := os.WriteFile(filePath, []byte("package test\n"), 0644)
		require.NoError(t, err)

		fragments := map[string][]CodeFragment{
			filePath: {
				{Content: "func NewFunc() int { return 42 }", Overwrite: false},
			},
		}

		err = InsertCodeFragments(fragments)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "func NewFunc()")
	})

	t.Run("creates file if not exists", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "new.go")

		fragments := map[string][]CodeFragment{
			filePath: {
				{Content: "func Auto() {}", Overwrite: false},
			},
		}

		err := InsertCodeFragments(fragments)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "package main")
		require.Contains(t, string(content), "func Auto()")
	})

	t.Run("multiple fragments in one file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "multi.go")

		err := os.WriteFile(filePath, []byte("package multi\n"), 0644)
		require.NoError(t, err)

		fragments := map[string][]CodeFragment{
			filePath: {
				{Content: "func A() {}", Overwrite: false},
				{Content: "func B() {}", Overwrite: false},
			},
		}

		err = InsertCodeFragments(fragments)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "func A()")
		require.Contains(t, string(content), "func B()")
	})
}

// --- FormatCodeAndFixImports tests ---

func TestFormatCodeAndFixImports(t *testing.T) {
	t.Run("formats valid go file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "format.go")

		// Unformatted code
		code := `package test

func   Hello()   string{
return "hello"
}
`
		err := os.WriteFile(filePath, []byte(code), 0644)
		require.NoError(t, err)

		err = FormatCodeAndFixImports(filePath)
		require.NoError(t, err)

		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), "func Hello() string")
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		err := FormatCodeAndFixImports("/nonexistent/path.go")
		require.Error(t, err)
	})

	t.Run("returns error for invalid go code", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "invalid.go")

		err := os.WriteFile(filePath, []byte("this is not valid go code"), 0644)
		require.NoError(t, err)

		err = FormatCodeAndFixImports(filePath)
		require.Error(t, err)
	})
}

// --- isTestFunction / isBenchmarkFunction tests ---

func TestIsTestFunction(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"TestFoo", true},
		{"TestA", true},
		{"Test", false},          // "Test" alone is not a test function (len <= 4)
		{"Testfoo", false},       // lowercase after Test
		{"TestFooBar", true},
		{"NotATest", false},
		{"", false},
		{"Testing", false},       // 'i' at position 4 is lowercase
		{"TestÜber", false},      // multi-byte char, byte comparison fails
		{"BenchmarkFoo", false},  // not a test function
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, isTestFunction(tt.name))
		})
	}
}

func TestIsBenchmarkFunction(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"BenchmarkFoo", true},
		{"BenchmarkA", true},
		{"Benchmark", false},      // len <= 9
		{"Benchmarkfoo", false},   // lowercase after Benchmark
		{"BenchmarkFooBar", true},
		{"NotABenchmark", false},
		{"", false},
		{"TestFoo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, isBenchmarkFunction(tt.name))
		})
	}
}

// --- ParseString edge cases ---

func TestParseStringEdgeCases(t *testing.T) {
	t.Run("embedded struct", func(t *testing.T) {
		code := `
		package test
		type Base struct {
			ID int
		}
		type Child struct {
			Base
			Name string
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		child := parsed.Struct("Child")
		require.Equal(t, "Child", child.Name)
		// Embedded field has no name, so name should be empty
		found := false
		for _, f := range child.Fields {
			if f.Type == "Base" && f.Name == "" {
				found = true
			}
		}
		require.True(t, found, "expected embedded Base field with empty name")
	})

	t.Run("empty struct", func(t *testing.T) {
		code := `
		package test
		type Empty struct {}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		empty := parsed.Struct("Empty")
		require.Equal(t, "Empty", empty.Name)
		require.Len(t, empty.Fields, 0)
	})

	t.Run("empty interface", func(t *testing.T) {
		code := `
		package test
		type Any interface {}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		any := parsed.Interface("Any")
		require.Equal(t, "Any", any.Name)
		require.Len(t, any.Methods, 0)
	})

	t.Run("variadic function", func(t *testing.T) {
		code := `
		package test
		func Variadic(args ...string) {}
		`
		output, err := ParseString(code)
		require.NoError(t, err)
		require.Len(t, output.Packages[0].Functions, 1)
		fn := output.Packages[0].Functions[0]
		require.Equal(t, "Variadic", fn.Name)
		require.Len(t, fn.Params, 1)
		require.Equal(t, "...string", fn.Params[0].Type)
	})

	t.Run("multiple return values", func(t *testing.T) {
		code := `
		package test
		func Multi() (int, string, error) { return 0, "", nil }
		`
		output, err := ParseString(code)
		require.NoError(t, err)
		fn := output.Packages[0].Functions[0]
		require.Len(t, fn.Returns, 3)
		require.Equal(t, "int", fn.Returns[0].Type)
		require.Equal(t, "string", fn.Returns[1].Type)
		require.Equal(t, "error", fn.Returns[2].Type)
	})

	t.Run("named return values", func(t *testing.T) {
		code := `
		package test
		func Named() (result string, err error) { return }
		`
		output, err := ParseString(code)
		require.NoError(t, err)
		fn := output.Packages[0].Functions[0]
		require.Len(t, fn.Returns, 2)
		require.Equal(t, "result", fn.Returns[0].Name)
		require.Equal(t, "string", fn.Returns[0].Type)
		require.Equal(t, "err", fn.Returns[1].Name)
		require.Equal(t, "error", fn.Returns[1].Type)
	})

	t.Run("exported flags", func(t *testing.T) {
		code := `
		package test
		type ExportedStruct struct {}
		type unexportedStruct struct {}
		func ExportedFunc() {}
		func unexportedFunc() {}

		type ExportedIface interface {
			ExportedMethod()
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])

		require.True(t, parsed.Struct("ExportedStruct").IsExported)
		require.False(t, parsed.Struct("unexportedStruct").IsExported)

		require.True(t, parsed.Function("ExportedFunc").IsExported)
		require.False(t, parsed.Function("unexportedFunc").IsExported)

		iface := parsed.Interface("ExportedIface")
		require.True(t, iface.IsExported)
		require.Len(t, iface.Methods, 1)
		require.True(t, iface.Methods[0].IsExported)
	})

	t.Run("test and benchmark function flags", func(t *testing.T) {
		code := `
		package test
		import "testing"
		func TestSomething(t *testing.T) {}
		func BenchmarkSomething(b *testing.B) {}
		func RegularFunc() {}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		for _, fn := range output.Packages[0].Functions {
			switch fn.Name {
			case "TestSomething":
				require.True(t, fn.IsTest)
				require.False(t, fn.IsBenchmark)
			case "BenchmarkSomething":
				require.False(t, fn.IsTest)
				require.True(t, fn.IsBenchmark)
			case "RegularFunc":
				require.False(t, fn.IsTest)
				require.False(t, fn.IsBenchmark)
			}
		}
	})

	t.Run("constants and variables", func(t *testing.T) {
		code := `
		package test
		const Pi = 3.14
		const (
			A = "alpha"
			B = "beta"
		)
		var GlobalVar string
		var (
			X int
			Y float64
		)
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		pkg := output.Packages[0]
		require.GreaterOrEqual(t, len(pkg.Constants), 3)
		require.GreaterOrEqual(t, len(pkg.Variables), 3)

		parsed := newHelper(&pkg)
		require.Equal(t, "3.14", parsed.Constant("Pi").Value)
		require.Equal(t, `"alpha"`, parsed.Constant("A").Value)
		require.Equal(t, `"beta"`, parsed.Constant("B").Value)

		require.Equal(t, "string", parsed.Variable("GlobalVar").Type)
		require.Equal(t, "int", parsed.Variable("X").Type)
		require.Equal(t, "float64", parsed.Variable("Y").Type)
	})

	t.Run("method with value receiver", func(t *testing.T) {
		code := `
		package test
		type Foo struct{}
		func (f Foo) Bar() string { return "" }
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		foo := parsed.Struct("Foo")
		require.Len(t, foo.Methods, 1)
		require.Equal(t, "Bar", foo.Methods[0].Name)
		require.Equal(t, "Foo", foo.Methods[0].Receiver)
	})

	t.Run("method with pointer receiver", func(t *testing.T) {
		code := `
		package test
		type Foo struct{}
		func (f *Foo) Baz() string { return "" }
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		foo := parsed.Struct("Foo")
		require.Len(t, foo.Methods, 1)
		require.Equal(t, "Baz", foo.Methods[0].Name)
		require.Equal(t, "*Foo", foo.Methods[0].Receiver)
	})

	t.Run("interface definition capture", func(t *testing.T) {
		code := `
		package test
		type Reader interface {
			Read(p []byte) (n int, err error)
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		reader := parsed.Interface("Reader")
		require.Contains(t, reader.Definition, "type Reader interface")
		require.Contains(t, reader.Definition, "Read")
	})

	t.Run("function definition capture", func(t *testing.T) {
		code := `
		package test
		func Add(a int, b int) int { return a + b }
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		fn := output.Packages[0].Functions[0]
		require.Contains(t, fn.Definition, "func Add(a int, b int) int")
	})

	t.Run("method definition capture", func(t *testing.T) {
		code := `
		package test
		type Calc struct{}
		func (c *Calc) Add(a int, b int) int { return a + b }
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		calc := parsed.Struct("Calc")
		require.Len(t, calc.Methods, 1)
		require.Contains(t, calc.Methods[0].Definition, "func (*Calc) Add(a int, b int) int")
	})

	t.Run("struct with tags", func(t *testing.T) {
		code := "package test\ntype Tagged struct {\n\tName string `json:\"name\" xml:\"name\"`\n}\n"
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		tagged := parsed.Struct("Tagged")
		f := tagged.Field("Name")
		require.Contains(t, f.Tag, `json:"name"`)
		require.Contains(t, f.Tag, `xml:"name"`)
	})

	t.Run("map type field", func(t *testing.T) {
		code := `
		package test
		type MapStruct struct {
			Data map[string]int
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		ms := parsed.Struct("MapStruct")
		f := ms.Field("Data")
		require.Equal(t, "map[string]int", f.Type)
		require.True(t, f.TypeDetails.IsMap)
	})

	t.Run("nested pointer slice", func(t *testing.T) {
		code := `
		package test
		type Container struct {
			Items []*string
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		c := parsed.Struct("Container")
		f := c.Field("Items")
		require.Equal(t, "[]*string", f.Type)
		require.True(t, f.Slice)
		require.False(t, f.Pointer)
	})

	t.Run("invalid go code returns error", func(t *testing.T) {
		code := `this is not valid go code`
		_, err := ParseString(code)
		require.Error(t, err)
	})
}

// --- parseDeclarationsFromCodeFrament edge cases ---

func TestParseDeclarationsFromCodeFragment(t *testing.T) {
	t.Run("code with package declaration", func(t *testing.T) {
		code := `package mypackage
		func Foo() {}
		`
		decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
		require.NoError(t, err)
		require.Len(t, decls, 1)
		require.Equal(t, "Foo", getDeclName(decls[0]))
	})

	t.Run("code without package declaration", func(t *testing.T) {
		code := `func Bar() {}`
		decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
		require.NoError(t, err)
		require.Len(t, decls, 1)
		require.Equal(t, "Bar", getDeclName(decls[0]))
	})

	t.Run("method declaration", func(t *testing.T) {
		code := `func (s *MyStruct) Method() string { return "" }`
		decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
		require.NoError(t, err)
		require.Len(t, decls, 1)
		require.Equal(t, "Method", getDeclName(decls[0]))
	})
}

// --- getDeclName edge cases ---

func TestGetDeclName(t *testing.T) {
	t.Run("returns empty for nil-like inputs", func(t *testing.T) {
		code := `
		import "fmt"
		`
		decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
		require.NoError(t, err)
		// Import declarations return empty name
		for _, d := range decls {
			name := getDeclName(d)
			require.Equal(t, "", name)
		}
	})
}

// --- getReceiverType tests ---

func TestGetReceiverType(t *testing.T) {
	t.Run("pointer receiver", func(t *testing.T) {
		decls, err := parseDeclarationsFromCodeFrament(CodeFragment{
			Content: `func (f *Foo) Bar() {}`,
		})
		require.NoError(t, err)
		require.Len(t, decls, 1)
		fd, ok := decls[0].(*ast.FuncDecl)
		require.True(t, ok)
		recv := getReceiverType(fd)
		require.Equal(t, "Foo", recv)
	})

	t.Run("value receiver", func(t *testing.T) {
		decls, err := parseDeclarationsFromCodeFrament(CodeFragment{
			Content: `func (f Foo) Bar() {}`,
		})
		require.NoError(t, err)
		require.Len(t, decls, 1)
		fd, ok := decls[0].(*ast.FuncDecl)
		require.True(t, ok)
		recv := getReceiverType(fd)
		require.Equal(t, "Foo", recv)
	})

	t.Run("no receiver", func(t *testing.T) {
		decls, err := parseDeclarationsFromCodeFrament(CodeFragment{
			Content: `func Bar() {}`,
		})
		require.NoError(t, err)
		require.Len(t, decls, 1)
		fd, ok := decls[0].(*ast.FuncDecl)
		require.True(t, ok)
		recv := getReceiverType(fd)
		require.Equal(t, "", recv)
	})
}
