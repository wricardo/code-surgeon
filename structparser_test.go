package codesurgeon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDirectory(t *testing.T) {
	tmp, err := ParseDirectory("./examples/structparser")
	require.NoError(t, err)

	for _, p := range tmp.Packages {
		if p.Package != "structs" {
			continue
		}
		parsed := newHelper(&p)

		t.Run("FirstStruct", func(t *testing.T) {
			firstStruct := parsed.Struct("FirstStruct")
			require.Len(t, firstStruct.Docs, 2)
			require.Equal(t, "FirstStruct this is the comment for the first struct.", firstStruct.Docs[0])
			require.Equal(t, "This is new line.", firstStruct.Docs[1])

			var f Field

			f = firstStruct.Field("Int")
			require.Equal(t, "Int", f.Name)
			require.Equal(t, "int", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)
			require.Equal(t, `json:"int" bson:"int"`, f.Tag)

			f = firstStruct.Field("Int8")
			require.Equal(t, "Int8", f.Name)
			require.Equal(t, "int8", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)
			require.Equal(t, `bson:"int8"`, f.Tag)

			f = firstStruct.Field("Int16")
			require.Equal(t, "Int16", f.Name)
			require.Equal(t, "int16", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Int32")
			require.Equal(t, "Int32", f.Name)
			require.Equal(t, "int32", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Int64")
			require.Equal(t, "Int64", f.Name)
			require.Equal(t, "int64", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Uint")
			require.Equal(t, "Uint", f.Name)
			require.Equal(t, "uint", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Uintptr")
			require.Equal(t, "Uintptr", f.Name)
			require.Equal(t, "uintptr", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Uint8")
			require.Equal(t, "Uint8", f.Name)
			require.Equal(t, "uint8", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Uint16")
			require.Equal(t, "Uint16", f.Name)
			require.Equal(t, "uint16", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Uint32")
			require.Equal(t, "Uint32", f.Name)
			require.Equal(t, "uint32", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Uint64")
			require.Equal(t, "Uint64", f.Name)
			require.Equal(t, "uint64", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Float32")
			require.Equal(t, "Float32", f.Name)
			require.Equal(t, "float32", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Float64")
			require.Equal(t, "Float64", f.Name)
			require.Equal(t, "float64", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Complex64")
			require.Equal(t, "Complex64", f.Name)
			require.Equal(t, "complex64", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Complex128")
			require.Equal(t, "Complex128", f.Name)
			require.Equal(t, "complex128", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Byte")
			require.Equal(t, "Byte", f.Name)
			require.Equal(t, "byte", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Rune")
			require.Equal(t, "Rune", f.Name)
			require.Equal(t, "rune", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("String")
			require.Equal(t, "String", f.Name)
			require.Equal(t, "string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("SecondStruct")
			require.Equal(t, "SecondStruct", f.Name)
			require.Equal(t, "SecondStruct", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("ArrayInt")
			require.Equal(t, "ArrayInt", f.Name)
			require.Equal(t, "[]int", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, true, f.Slice)

			f = firstStruct.Field("SliceString")
			require.Equal(t, "SliceString", f.Name)
			require.Equal(t, "[]string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, true, f.Slice)

			f = firstStruct.Field("SlicePointerString")
			require.Equal(t, "SlicePointerString", f.Name)
			require.Equal(t, "[]*string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, true, f.Slice)

			f = firstStruct.Field("PointerSliceString")
			require.Equal(t, "PointerSliceString", f.Name)
			require.Equal(t, "*[]string", f.Type)
			require.Equal(t, true, f.Pointer)
			require.Equal(t, true, f.Slice)

			f = firstStruct.Field("PointerSlicePointerString")
			require.Equal(t, "PointerSlicePointerString", f.Name)
			require.Equal(t, "*[]*string", f.Type)
			require.Equal(t, true, f.Pointer)
			require.Equal(t, true, f.Slice)

			f = firstStruct.Field("ChanString")
			require.Equal(t, "ChanString", f.Name)
			require.Equal(t, "chan string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("RChanString")
			require.Equal(t, "RChanString", f.Name)
			require.Equal(t, "<-chan string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("SChanString")
			require.Equal(t, "SChanString", f.Name)
			require.Equal(t, "chan<- string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("MapStringString")
			require.Equal(t, "MapStringString", f.Name)
			require.Equal(t, "map[string]string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("MapPointerStringString")
			require.Equal(t, "MapPointerStringString", f.Name)
			require.Equal(t, "map[*string]string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("MapPointerStringPointerString")
			require.Equal(t, "MapPointerStringPointerString", f.Name)
			require.Equal(t, "map[*string]*string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("PointerMapStringString")
			require.Equal(t, "PointerMapStringString", f.Name)
			require.Equal(t, "*map[string]string", f.Type)
			require.Equal(t, true, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("PointerMapPointerStringPointerString")
			require.Equal(t, "PointerMapPointerStringPointerString", f.Name)
			require.Equal(t, "*map[*string]*string", f.Type)
			require.Equal(t, true, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("Func")
			require.Equal(t, "Func", f.Name)
			require.Equal(t, "SomeFunc", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("PointerFunc")
			require.Equal(t, "PointerFunc", f.Name)
			require.Equal(t, "*SomeFunc", f.Type)
			require.Equal(t, true, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("MapStringSliceString")
			require.Equal(t, "MapStringSliceString", f.Name)
			require.Equal(t, "map[string][]string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("MapStringSlicePointerString")
			require.Equal(t, "MapStringSlicePointerString", f.Name)
			require.Equal(t, "map[string][]*string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("MapPointerStringSlicePointerString")
			require.Equal(t, "MapPointerStringSlicePointerString", f.Name)
			require.Equal(t, "map[*string][]*string", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("MapChanPointerStringStruct")
			require.Equal(t, "MapChanPointerStringStruct", f.Name)
			require.Equal(t, "map[chan *string]SecondStruct", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("SpecialString")
			require.Equal(t, "SpecialString", f.Name)
			require.Equal(t, "SpecialString", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("PackageStruct")
			require.Equal(t, "PackageStruct", f.Name)
			require.Equal(t, "other.Struct", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("PointerPackageStruct")
			require.Equal(t, "PointerPackageStruct", f.Name)
			require.Equal(t, "*other.Struct", f.Type)
			require.Equal(t, true, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("SlicePointerPackageStruct")
			require.Equal(t, "SlicePointerPackageStruct", f.Name)
			require.Equal(t, "[]*other.Struct", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, true, f.Slice)

			f = firstStruct.Field("MapStringPackageStruct")
			require.Equal(t, "MapStringPackageStruct", f.Name)
			require.Equal(t, "map[string]other.Struct", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)

			f = firstStruct.Field("ChanPackagePointerStruct")
			require.Equal(t, "ChanPackagePointerStruct", f.Name)
			require.Equal(t, "chan *other.Struct", f.Type)
			require.Equal(t, false, f.Pointer)
			require.Equal(t, false, f.Slice)
		})

		t.Run("FirstStruct", func(t *testing.T) {
			tmp := parsed.Struct("CommentsAndDocs")

			require.Len(t, tmp.Docs, 1)
			require.Equal(t, "CommentsAndDocs this is the comment for the CommentsAndDocs struct.", tmp.Docs[0])

			t.Run("fields comments and docs", func(t *testing.T) {
				{
					docs := tmp.Field("SingleDoc").Docs
					require.Len(t, docs, 1)
					require.Equal(t, "this is line 1 of comment 001", docs[0])
				}
				{
					docs := tmp.Field("MultiLineDoc").Docs
					require.Len(t, docs, 2)
					require.Equal(t, "this is line 1 of comment 001", docs[0])
					require.Equal(t, "this is line 2 of comment 002", docs[1])
				}
				{
					docs := tmp.Field("MixedSpacesDoc").Docs
					require.Len(t, docs, 2)
					require.Equal(t, "this is line 1 of comment 003", docs[0])
					require.Equal(t, "this is line 2 of comment 004", docs[1])
				}
				{
					docs := tmp.Field("MixedTypesDoc").Docs
					require.Len(t, docs, 2)
					require.Equal(t, "this is line 1 of comment 005", docs[0])
					require.Equal(t, "this is line 2 of comment 006", docs[1])
				}
				{
					docs := tmp.Field("DocAndComment").Docs
					com := tmp.Field("DocAndComment").Comment
					require.Len(t, docs, 1)
					require.Equal(t, "this is line 1 of comment 007", docs[0])
					require.Equal(t, "comment 008", com)
				}
				{
					docs := tmp.Field("CommentNoSpaces").Docs
					com := tmp.Field("CommentNoSpaces").Comment
					require.Len(t, docs, 0)
					require.Equal(t, "comment abc", com)
				}
				{
					docs := tmp.Field("StarDoc").Docs
					com := tmp.Field("StarDoc").Comment
					require.Len(t, docs, 1)
					require.Equal(t, "this is line 1 of comment 009", docs[0])
					require.Equal(t, "comment 010", com)
				}
				{
					docs := tmp.Field("CommentWithTag").Docs
					com := tmp.Field("CommentWithTag").Comment
					require.Len(t, docs, 1)
					require.Equal(t, "this is line 1 of comment 010", docs[0])
					require.Equal(t, "comment 11", com)
				}
				{
					docs := tmp.Field("CrazyDoc").Docs
					require.Len(t, docs, 9)
					require.Equal(t, "001", docs[0])
					require.Equal(t, "002", docs[1])
					require.Equal(t, "003", docs[2])
					require.Equal(t, "004", docs[3])
					require.Equal(t, "005", docs[4])
					require.Equal(t, "006", docs[5])
					require.Equal(t, "007", docs[6])
					require.Equal(t, "* 008 *", docs[7])
					require.Equal(t, "009", docs[8])
				}
			})
		})

		t.Run("SecondStruct", func(t *testing.T) {
			secondStruct := parsed.Struct("SecondStruct")
			require.Len(t, secondStruct.Docs, 0)
		})

		// New test cases for Variables
		t.Run("Variables", func(t *testing.T) {
			variable := parsed.Variable("MyVariable")
			require.Equal(t, "MyVariable", variable.Name)
			require.Equal(t, "string", variable.Type)
		})

		// New test cases for Constants
		t.Run("Constants", func(t *testing.T) {
			constant := parsed.Constant("MyConstant")
			require.Equal(t, "MyConstant", constant.Name)
			require.Equal(t, `"world"`, constant.Value) // Example value
		})

		// New test cases for Functions
		t.Run("Functions", func(t *testing.T) {
			function := parsed.Function("MyFunction")
			require.Equal(t, "MyFunction", function.Name)
			require.Equal(t, 1, len(function.Params))
			require.Equal(t, "string", function.Params[0].Type)
			require.Equal(t, 2, len(function.Returns))
			require.Equal(t, "string", function.Returns[0].Type)
			require.Equal(t, "error", function.Returns[1].Type)
		})

		// New test case for Package Name
		t.Run("PackageName", func(t *testing.T) {
			require.Equal(t, "structs", tmp.Packages[0].Package)
		})

		// New test case for Imports
		t.Run("Imports", func(t *testing.T) {
			require.Contains(t, tmp.Packages[0].Imports, "context")
			require.Contains(t, tmp.Packages[0].Imports, "github.com/wricardo/code-surgeon/examples/structparser/other")
			require.Contains(t, tmp.Packages[0].Imports, "time")
		})
	}
}

func TestCleanDocText(t *testing.T) {
	inputs := []struct {
		input  string
		expect string
	}{
		{input: "// something", expect: "something"},
		{input: "//   something  ", expect: "something"},
		{input: "//something", expect: "something"},
		{input: "///something", expect: "/something"},
		{input: "/*something*/", expect: "something"},
		{input: "/* something */", expect: "something"},
		{input: "/*  something  */", expect: "something"},
		{input: "/*something */", expect: "something"},
		{input: "/*  something*/", expect: "something"},
		{input: "/*/ something*/", expect: "/ something"},
		{input: "/*/ something /*/", expect: "/ something /"},
		{input: "\nsomething\n", expect: "something"},
	}
	for _, v := range inputs {
		require.Equal(t, v.expect, cleanDocText(v.input))
	}
}

func TestFirstStructMethods(t *testing.T) {
	tmp, err := ParseDirectory("./examples/structparser/")
	require.NoError(t, err)

	for _, p := range tmp.Packages {
		if p.Package == "structs" {
			parsed := newHelper(&p)

			t.Run("FirstStruct", func(t *testing.T) {
				firstStruct := parsed.Struct("FirstStruct")
				require.Len(t, firstStruct.Docs, 2)
				require.Equal(t, "FirstStruct this is the comment for the first struct.", firstStruct.Docs[0])
				require.Len(t, firstStruct.Methods, 3)
				for _, m := range firstStruct.Methods {
					if m.Name == "MyTestMethod" {
						require.Equal(t, "MyTestMethod(ctx context.Context, x []string, y []string, z int) (a string, b string, c int)", m.Signature)
						require.Len(t, m.Docs, 1)
						require.Equal(t, "MyTestMethod this is the comment for the MyTestMethod method.", m.Docs[0])
					} else if m.Name == "MyOtherTestMethod" {
						require.Equal(t, "MyOtherTestMethod(ctx context.Context, x string) (string, error)", m.Signature)
						require.Len(t, m.Docs, 1)
						require.Equal(t, "MyOtherTestMethod this is the comment for the MyOtherTestMethod method.", m.Docs[0])
					}
				}
			})
		}
	}
}

func TestPrivateStruct(t *testing.T) {
	tmp, err := ParseDirectory("./examples/structparser/")
	require.NoError(t, err)

	for _, p := range tmp.Packages {
		if p.Package == "structs" {
			parsed := newHelper(&p)

			t.Run("privateStruct", func(t *testing.T) {
				privateStruct := parsed.Struct("privateStruct")

				f := privateStruct.Field("String")
				require.Equal(t, "String", f.Name)
				require.Equal(t, "string", f.Type)
				require.Equal(t, false, f.Pointer)
				require.Equal(t, false, f.Slice)
				require.Empty(t, f.Tag)

				require.Len(t, privateStruct.Methods, 1)
				require.Equal(t, "MyPrivateStructMethod(ctx context.Context, x string) (string, error)", privateStruct.Methods[0].Signature)
			})
		}
	}
}

func TestParseString(t *testing.T) {
	// Test simple struct parsing
	t.Run("SimpleStruct", func(t *testing.T) {
		code := `
		package test
		// SimpleStruct represents a simple test case
		type SimpleStruct struct {
			Name  string  // Name is a string field
			Value int     // Value is an integer field
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		structInfo := parsed.Struct("SimpleStruct")

		require.Len(t, structInfo.Docs, 1)
		require.Equal(t, "SimpleStruct represents a simple test case", structInfo.Docs[0])

		var f Field
		f = structInfo.Field("Name")
		require.Equal(t, "Name", f.Name)
		require.Equal(t, "string", f.Type)
		require.Equal(t, false, f.Pointer)
		require.Equal(t, false, f.Slice)
		require.Equal(t, "Name is a string field", f.Comment)

		f = structInfo.Field("Value")
		require.Equal(t, "Value", f.Name)
		require.Equal(t, "int", f.Type)
		require.Equal(t, false, f.Pointer)
		require.Equal(t, false, f.Slice)
		require.Equal(t, "Value is an integer field", f.Comment)
	})

	t.Run("Two structs", func(t *testing.T) {
		code := `
		package test
			type (
			SimpleStruct struct {
				Name  string  // Name is a string field
				Value int     // Value is an integer field
			}

			OtherStruct struct {
				Name  string  // Name is a string field
				Value int     // Value is an integer field
			}
		)
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		structInfo := parsed.Struct("SimpleStruct")

		var f Field
		f = structInfo.Field("Name")
		require.Equal(t, "Name", f.Name)
		require.Equal(t, "string", f.Type)
		require.Equal(t, false, f.Pointer)
		require.Equal(t, false, f.Slice)
		require.Equal(t, "Name is a string field", f.Comment)

		f = structInfo.Field("Value")
		require.Equal(t, "Value", f.Name)
		require.Equal(t, "int", f.Type)
		require.Equal(t, false, f.Pointer)
		require.Equal(t, false, f.Slice)
		require.Equal(t, "Value is an integer field", f.Comment)

		structInfo = parsed.Struct("OtherStruct")

		require.Len(t, structInfo.Docs, 0)

		f = structInfo.Field("Name")
		require.Equal(t, "Name", f.Name)
		require.Equal(t, "string", f.Type)
		require.Equal(t, false, f.Pointer)
		require.Equal(t, false, f.Slice)
		require.Equal(t, "Name is a string field", f.Comment)

		f = structInfo.Field("Value")
		require.Equal(t, "Value", f.Name)
		require.Equal(t, "int", f.Type)
	})

	// Test parsing methods of a struct
	t.Run("StructWithMethods", func(t *testing.T) {
		code := `
		package test
		// StructWithMethods represents a struct with methods
		type StructWithMethods struct {}

		// Greet returns a greeting message
		func (s *StructWithMethods) Greet(name string) string {
			return "Hello " + name
		}

		// Sum adds two integers
		func (s *StructWithMethods) Sum(a, b int) int {
			return a + b
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		structInfo := parsed.Struct("StructWithMethods")

		require.Len(t, structInfo.Methods, 2)

		require.Equal(t, "Greet(name string) string", structInfo.Methods[0].Signature)
		require.Equal(t, "Sum(a int, b int) int", structInfo.Methods[1].Signature)

		// Check method documentation
		require.Equal(t, "Greet returns a greeting message", structInfo.Methods[0].Docs[0])
		require.Equal(t, "Sum adds two integers", structInfo.Methods[1].Docs[0])
	})

	// Test struct with complex types
	t.Run("ComplexStruct", func(t *testing.T) {
		code := `
		package test
		// ComplexStruct represents a struct with various field types
		type ComplexStruct struct {
			SliceOfStrings []string
			PointerToInt   *int
			MapOfIntToStr  map[int]string
			FuncField      func(string) error
		}
		`
		output, err := ParseString(code)
		require.NoError(t, err)

		parsed := newHelper(&output.Packages[0])
		structInfo := parsed.Struct("ComplexStruct")

		f := structInfo.Field("SliceOfStrings")
		require.Equal(t, "SliceOfStrings", f.Name)
		require.Equal(t, "[]string", f.Type)
		require.Equal(t, false, f.Pointer)
		require.Equal(t, true, f.Slice)

		f = structInfo.Field("PointerToInt")
		require.Equal(t, "PointerToInt", f.Name)
		require.Equal(t, "*int", f.Type)
		require.Equal(t, true, f.Pointer)
		require.Equal(t, false, f.Slice)

		f = structInfo.Field("MapOfIntToStr")
		require.Equal(t, "MapOfIntToStr", f.Name)
		require.Equal(t, "map[int]string", f.Type)
		require.Equal(t, false, f.Pointer)
		require.Equal(t, false, f.Slice)

		f = structInfo.Field("FuncField")
		require.Equal(t, "FuncField", f.Name)
		require.Equal(t, "func(string) (error)", f.Type)
	})
}

func TestParseStringWithFunctions(t *testing.T) {
	// Test parsing top-level functions
	t.Run("TopLevelFunctions", func(t *testing.T) {
		code := `
		package main
		// HelloWorld returns a greeting message
		func HelloWorld() string {
			return "Hello, World!"
		}

		// Add sums two integers
		func Add(a, b int) int {
			return a + b
		}

		// Divide divides two floats and returns the result
		func Divide(x, y float64) (float64, error) {
			if y == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return x / y, nil
		}
		`

		output, err := ParseString(code)
		require.NoError(t, err)
		require.NotNil(t, output)

		countOk := 0
		require.Len(t, output.Packages[0].Functions, 3)

		for _, f := range output.Packages[0].Functions {
			switch f.Name {
			case "HelloWorld":
				require.Equal(t, "HelloWorld", f.Name)
				require.Equal(t, "HelloWorld() string", f.Signature)
				require.Equal(t, []string{"HelloWorld returns a greeting message"}, f.Docs)
				countOk++
			case "Add":
				require.Equal(t, "Add", f.Name)
				require.Equal(t, "Add(a int, b int) int", f.Signature)
				require.Equal(t, []string{"Add sums two integers"}, f.Docs)
				countOk++
			case "Divide":
				require.Equal(t, "Divide", f.Name)
				require.Equal(t, "Divide(x float64, y float64) (float64, error)", f.Signature)
				require.Equal(t, []string{"Divide divides two floats and returns the result"}, f.Docs)
				countOk++
			}
		}
		require.Equal(t, 3, countOk)
	})
}

func TestParseStringComments(t *testing.T) {
	// Test parsing comments from top-level functions
	t.Run("TopLevelFunctions", func(t *testing.T) {
		code := `
		package main

		// a is a string
		var a string

		// random line comment

		// HelloWorld returns a greeting message
		func HelloWorld() string {
			return "Hello, World!"
		}


		// bottom
		// bottom2

		`

		// fset := token.NewFileSet()

		// // Parse the Go code into an abstract syntax tree (AST).
		// node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
		// if err != nil {
		// 	panic(err)
		// }

		// // Iterate through all the declarations in the parsed file.
		// for _, decl := range node.Decls {
		// 	// Check if the declaration is a function declaration.
		// 	if funcDecl, ok := decl.(*ast.FuncDecl); ok {
		// 		// Get the function name.
		// 		funcName := funcDecl.Name.Name

		// 		// Get the comments associated with the function.
		// 		var comments []string
		// 		if funcDecl.Doc != nil {
		// 			for _, comment := range funcDecl.Doc.List {
		// 				// Trim the leading "// " or "/* " from the comment text.
		// 				trimmedComment := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		// 				comments = append(comments, trimmedComment)
		// 			}
		// 		}

		// 		fmt.Println("Function:", funcName)
		// 		fmt.Println("Comments:", comments)
		// 	}
		// }

		output, err := ParseString(code)
		require.NoError(t, err)
		require.NotNil(t, output)
		pkg := output.Packages[0]

		require.Len(t, pkg.Functions, 1)

		// Test first function (HelloWorld)
		helloWorldFunc := pkg.Functions[0]
		require.Equal(t, "HelloWorld", helloWorldFunc.Name)
		require.Equal(t, []string{"HelloWorld returns a greeting message"}, helloWorldFunc.Docs)
	})
}

func TestParseInterface(t *testing.T) {
	code := `
	package test
	// Greeter is an interface for greeting
	type Greeter interface {
		// Greet returns a greeting message
		Greet(name string) string
		// Farewell says goodbye
		Farewell() error
	}
	`
	output, err := ParseString(code)
	require.NoError(t, err, "Parsing code should not result in an error")
	require.NotNil(t, output, "Parsed output should not be nil")

	parsed := newHelper(&output.Packages[0])
	iface := parsed.Interface("Greeter")

	require.Equal(t, "Greeter", iface.Name, "Interface name should be 'Greeter'")

	// Check the interface documentation
	t.Run("Interface Documentation", func(t *testing.T) {
		require.Len(t, iface.Docs, 1, "Greeter interface should have 1 doc line")
		require.Equal(t, "Greeter is an interface for greeting", iface.Docs[0], "Interface documentation mismatch")
	})

	// Check methods
	require.Len(t, iface.Methods, 2, "Greeter interface should have 2 methods")

	t.Run("Method Greet", func(t *testing.T) {
		method := iface.Methods[0]
		require.Equal(t, "Greet", method.Name, "Method name should be 'Greet'")
		require.Len(t, method.Params, 1, "Greet method should have 1 parameter")
		require.Equal(t, "name", method.Params[0].Name, "Parameter name should be 'name'")
		require.Equal(t, "string", method.Params[0].Type, "Parameter type should be 'string'")
		require.Len(t, method.Returns, 1, "Greet method should have 1 return type")
		require.Equal(t, "string", method.Returns[0].Type, "Return type should be 'string'")
		require.Equal(t, "Greet(name string) (string)", method.Signature, "Method signature mismatch")
		require.Len(t, method.Docs, 1, "Greet method should have 1 doc line")
		require.Equal(t, "Greet returns a greeting message", method.Docs[0], "Method documentation mismatch")
	})

	t.Run("Method Farewell", func(t *testing.T) {
		method := iface.Methods[1]
		require.Equal(t, "Farewell", method.Name, "Method name should be 'Farewell'")
		require.Len(t, method.Params, 0, "Farewell method should have no parameters")
		require.Len(t, method.Returns, 1, "Farewell method should have 1 return type")
		require.Equal(t, "error", method.Returns[0].Type, "Return type should be 'error'")
		require.Equal(t, "Farewell() (error)", method.Signature, "Method signature mismatch")
		require.Len(t, method.Docs, 1, "Farewell method should have 1 doc line")
		require.Equal(t, "Farewell says goodbye", method.Docs[0], "Method documentation mismatch")
	})
}

func TestParseFunctionWithBody(t *testing.T) {
	code := `
	package test
	// HelloWorld returns a greeting message
	func HelloWorld() string {
		return "Hello, World!"
	}
	`
	output, err := ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Len(t, output.Packages[0].Functions, 1)
	require.Equal(t, "HelloWorld", output.Packages[0].Functions[0].Name)
	require.Equal(t, "HelloWorld() string", output.Packages[0].Functions[0].Signature)
	require.Equal(t, []string{"HelloWorld returns a greeting message"}, output.Packages[0].Functions[0].Docs)
	require.NotEmpty(t, output.Packages[0].Functions[0].Body)
	require.Contains(t, output.Packages[0].Functions[0].Body, "return \"Hello, World!\"")
}

func TestParseMethodsWithBody(t *testing.T) {
	code := `
	package test
	// Greeter is an interface for greeting
	type Greeter interface {
		// Greet returns a greeting message
		Greet(name string) string
		// Farewell says goodbye
		Farewell() error
	}

	// MyGreeter is a Greeter implementation
	type MyGreeter struct {}

	// Greet returns a greeting message
	func (g *MyGreeter) Greet(name string) string {
		return "Hello, " + name
	}
	`
	output, err := ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, output)

	parsed := newHelper(&output.Packages[0])
	myGreeter := parsed.Struct("MyGreeter")

	require.Len(t, myGreeter.Methods, 1)

	t.Run("Method Greet", func(t *testing.T) {
		method := myGreeter.Methods[0]
		require.Equal(t, "Greet", method.Name)
		require.Equal(t, "Greet(name string) string", method.Signature)
		require.Len(t, method.Docs, 1)
		require.Equal(t, "Greet returns a greeting message", method.Docs[0])
		require.NotEmpty(t, method.Body)
		require.Contains(t, method.Body, "return \"Hello, \" + name")
	})
}

func TestParseGenericMethod(t *testing.T) {
	code := `
	package test

	// UnimplementedZivoAPIHandler returns CodeUnimplemented from all methods.
	type UnimplementedZivoAPIHandler struct{}

	func (UnimplementedZivoAPIHandler) SendSms(ctx context.Context, r *connect.Request[FirstStruct]) (*connect.Response[FirstStruct], error) {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("trinity.zivo.ZivoAPI.SendSms is not implemented"))
	}
	`
	parsed, err := ParseString(code)
	require.NoError(t, err)

	parsedHelper := newHelper(&parsed.Packages[0])
	structInfo := parsedHelper.Struct("UnimplementedZivoAPIHandler")
	require.NotNil(t, structInfo)

	require.Len(t, structInfo.Methods, 1)
	method := structInfo.Methods[0]

	expectedSignature := "SendSms(ctx context.Context, r *connect.Request[FirstStruct]) (*connect.Response[FirstStruct], error)"
	require.Equal(t, expectedSignature, method.Signature)

	require.Equal(t, "*connect.Request[FirstStruct]", structInfo.Methods[0].Params[1].Type)
	require.Equal(t, "*connect.Response[FirstStruct]", structInfo.Methods[0].Returns[0].Type)
}

/*
	package test

	type Person struct {
		Name string
	}

	func SomeFunc(strParam string, strPointer *string, p Person) (error, *Person) {
		return nil, nil
	}
*/

// test parsing the returns of SomeFunc
func TestSomeFuncReturns(t *testing.T) {
	code := `
	package test

	type Person struct {
		Name string
	}

	func SomeFunc(strParam string, strPointer *string, p Person) (error, *Person) {
		return nil, nil
	}
	`
	output, err := ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, output)

	parsed := newHelper(&output.Packages[0])
	someFunc := parsed.Function("SomeFunc")

	require.Len(t, someFunc.Returns, 2)

	require.Equal(t, "error", someFunc.Returns[0].Type)
	require.Equal(t, "*Person", someFunc.Returns[1].Type)
}
