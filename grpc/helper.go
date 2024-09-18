package grpc

import (
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/api"
)

// Helper function to convert parsed info to proto format
func convertParsedInfoToProto(parsedInfo *codesurgeon.ParsedInfo) []*api.Package {
	var packages []*api.Package
	for _, pkg := range parsedInfo.Packages {
		packages = append(packages, &api.Package{
			Package:    pkg.Package,
			Imports:    pkg.Imports,
			Structs:    convertStructsToProto(pkg.Structs),
			Functions:  convertFunctionsToProto(pkg.Functions),
			Variables:  convertVariablesToProto(pkg.Variables),
			Constants:  convertConstantsToProto(pkg.Constants),
			Interfaces: convertInterfacesToProto(pkg.Interfaces),
		})
	}
	return packages
}

// Helper function to convert structs from codesurgeon to proto
func convertStructsToProto(structs []codesurgeon.Struct) []*api.Struct {
	var protoStructs []*api.Struct
	for _, s := range structs {
		protoStruct := &api.Struct{
			Name:    s.Name,
			Fields:  convertFieldsToProto(s.Fields),
			Methods: convertMethodsToProto(s.Methods),
			Docs:    s.Docs,
		}
		protoStructs = append(protoStructs, protoStruct)
	}
	return protoStructs
}

// Helper function to convert fields from codesurgeon to proto
func convertFieldsToProto(fields []codesurgeon.Field) []*api.Field {
	var protoFields []*api.Field
	for _, f := range fields {
		protoField := &api.Field{
			Name:    f.Name,
			Type:    f.Type,
			Tag:     f.Tag,
			Private: f.Private,
			Pointer: f.Pointer,
			Slice:   f.Slice,
			Docs:    f.Docs,
			Comment: f.Comment,
		}
		protoFields = append(protoFields, protoField)
	}
	return protoFields
}

// Helper function to convert methods from codesurgeon to proto
func convertMethodsToProto(methods []codesurgeon.Method) []*api.Method {
	var protoMethods []*api.Method
	for _, m := range methods {
		protoMethod := &api.Method{
			Receiver:  m.Receiver,
			Name:      m.Name,
			Params:    convertParamsToProto(m.Params),
			Returns:   convertParamsToProto(m.Returns),
			Docs:      m.Docs,
			Signature: m.Signature,
			Body:      m.Body,
		}
		protoMethods = append(protoMethods, protoMethod)
	}
	return protoMethods
}

// Helper function to convert parameters from codesurgeon to proto
func convertParamsToProto(params []codesurgeon.Param) []*api.Param {
	var protoParams []*api.Param
	for _, p := range params {
		protoParam := &api.Param{
			Name: p.Name,
			Type: p.Type,
		}
		protoParams = append(protoParams, protoParam)
	}
	return protoParams
}

// Helper function to convert functions from codesurgeon to proto
func convertFunctionsToProto(functions []codesurgeon.Function) []*api.Function {
	var protoFunctions []*api.Function
	for _, f := range functions {
		protoFunction := &api.Function{
			Name:      f.Name,
			Params:    convertParamsToProto(f.Params),
			Returns:   convertParamsToProto(f.Returns),
			Docs:      f.Docs,
			Signature: f.Signature,
			Body:      f.Body,
		}
		protoFunctions = append(protoFunctions, protoFunction)
	}
	return protoFunctions
}

// Helper function to convert variables from codesurgeon to proto
func convertVariablesToProto(variables []codesurgeon.Variable) []*api.Variable {
	var protoVariables []*api.Variable
	for _, v := range variables {
		protoVariable := &api.Variable{
			Name: v.Name,
			Type: v.Type,
			Docs: v.Docs,
		}
		protoVariables = append(protoVariables, protoVariable)
	}
	return protoVariables
}

// Helper function to convert constants from codesurgeon to proto
func convertConstantsToProto(constants []codesurgeon.Constant) []*api.Constant {
	var protoConstants []*api.Constant
	for _, c := range constants {
		protoConstant := &api.Constant{
			Name:  c.Name,
			Value: c.Value,
			Docs:  c.Docs,
		}
		protoConstants = append(protoConstants, protoConstant)
	}
	return protoConstants
}

// Helper function to convert interfaces from codesurgeon to proto
func convertInterfacesToProto(interfaces []codesurgeon.Interface) []*api.Interface {
	var protoInterfaces []*api.Interface
	for _, i := range interfaces {
		protoInterface := &api.Interface{
			Name:    i.Name,
			Methods: convertMethodsToProto(i.Methods),
			Docs:    i.Docs,
		}
		protoInterfaces = append(protoInterfaces, protoInterface)
	}
	return protoInterfaces
}
