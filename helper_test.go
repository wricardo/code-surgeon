package codesurgeon

// helper is a struct that provides helper methods for accessing parsed data.
type helper struct {
	structs    map[string]helperField
	variables  map[string]Variable
	constants  map[string]Constant
	functions  map[string]Function
	incerfaces map[string]Interface
	output     *Package
}

// newHelper initializes a new helper struct from the parsed output.
func newHelper(out *Package) helper {
	h := helper{
		structs:    make(map[string]helperField),
		variables:  make(map[string]Variable),
		constants:  make(map[string]Constant),
		functions:  make(map[string]Function),
		incerfaces: make(map[string]Interface),
		output:     out,
	}

	// Populate structs
	for _, s := range out.Structs {
		h.structs[s.Name] = newHelperField(s.Fields, s)
	}

	// Populate variables
	for _, v := range out.Variables {
		h.variables[v.Name] = v
	}

	// Populate constants
	for _, c := range out.Constants {
		h.constants[c.Name] = c
	}

	// Populate functions
	for _, f := range out.Functions {
		h.functions[f.Name] = f
	}

	// Populate interfaces
	for _, i := range out.Interfaces {
		h.incerfaces[i.Name] = i
	}

	return h
}

// Struct returns the helperField for a given struct name.
func (h helper) Struct(name string) helperField {
	return h.structs[name]
}

// Variable returns a Variable by name.
func (h helper) Variable(name string) Variable {
	return h.variables[name]
}

// Constant returns a Constant by name.
func (h helper) Constant(name string) Constant {
	return h.constants[name]
}

// Function returns a Function by name.
func (h helper) Function(name string) Function {
	return h.functions[name]
}

func (h helper) Interface(name string) Interface {
	return h.incerfaces[name]
}

// helperField is a helper struct for handling fields within a struct.
type helperField struct {
	Struct
	fields map[string]Field
}

// newHelperField initializes a new helperField.
func newHelperField(fields []Field, structt Struct) helperField {
	hf := helperField{
		Struct: structt,
		fields: make(map[string]Field),
	}

	// Populate fields
	for _, f := range fields {
		hf.fields[f.Name] = f
	}

	return hf
}

// Field returns a Field by name from the helperField.
func (h helperField) Field(name string) Field {
	return h.fields[name]
}
