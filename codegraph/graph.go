// Package codegraph builds a typed graph of a Go code base (packages, types,
// functions, methods, fields and the relations between them) and stores it in
// an embedded bbolt database so it can be traversed without any external service.
package codegraph

// Node kinds.
const (
	KindPackage   = "Package"
	KindStruct    = "Struct"
	KindInterface = "Interface"
	KindType      = "Type" // any other named type (e.g. type ID string, type Handler func())
	KindFunc      = "Func"
	KindMethod    = "Method"
	KindField     = "Field"
)

// Relation names. Edges are directed From -> To.
const (
	RelBelongsTo   = "BELONGS_TO"   // declaration -> package
	RelHasMethod   = "HAS_METHOD"   // struct/type/interface -> method
	RelHasField    = "HAS_FIELD"    // struct -> field
	RelEmbeds      = "EMBEDS"       // struct/interface -> embedded type
	RelFieldType   = "FIELD_TYPE"   // field -> named type used by the field
	RelParamType   = "PARAM_TYPE"   // func/method -> named type used by a parameter
	RelReturnsType = "RETURNS_TYPE" // func/method -> named type used by a result
	RelCalls       = "CALLS"        // func/method (or package, for var initialisers) -> statically resolved callee
	RelReferences  = "REFERENCES"   // func/method (or package) -> func/method used as a value, not called
	RelImplements  = "IMPLEMENTS"   // struct/type -> interface (value or pointer receiver)
	RelImports     = "IMPORTS"      // package -> package
)

// Node is a vertex of the code graph.
type Node struct {
	ID       string `json:"id"`                 // stable id, e.g. "method:github.com/x/y.Calculator.Add"
	Kind     string `json:"kind"`               // one of the Kind* constants
	Name     string `json:"name"`               // "Calculator.Add" for methods/fields, plain name otherwise, import path for packages
	Short    string `json:"short,omitempty"`    // last name segment ("Add"); equals Name when there is no qualifier
	Pkg      string `json:"pkg,omitempty"`      // import path of the owning package
	File     string `json:"file,omitempty"`     // declaration file, relative to the build dir when possible
	Line     int    `json:"line,omitempty"`     // first line of the declaration name
	EndLine  int    `json:"end_line,omitempty"` // last line of the whole declaration (funcs, methods, types)
	Exported bool   `json:"exported,omitempty"` // exported identifier
	Ext      bool   `json:"ext,omitempty"`      // referenced but declared outside the loaded packages (stub node)
	Sig      string `json:"sig,omitempty"`      // signature for funcs/methods, type expression for fields
	Doc      string `json:"doc,omitempty"`      // doc comment
}

// Edge is a directed, typed relation between two nodes.
type Edge struct {
	From string `json:"from"`
	Rel  string `json:"rel"`
	To   string `json:"to"`
}

// Dir selects the traversal direction.
type Dir int

const (
	DirOut Dir = iota // follow edges From -> To
	DirIn             // follow edges To -> From
)
