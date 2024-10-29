package codesurgeon

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/rs/zerolog/log"
)

func PrettyPrint(
	parsed []*ParsedInfo,
	mode string,
	ignoreRules []string,
	plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, tags bool,
	comments bool,
) string {
	var sb strings.Builder

	switch mode {
	case "json":
		return prettyPrintJSON(parsed)
	case "grepindex":
		return prettyPrintGrepIndex(parsed, ignoreRules, &sb)
	case "llm":
		return prettyPrintLLM(parsed, ignoreRules, plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, comments, &sb)
	case "text_short", "":
		return prettyPrintTextShort(parsed, ignoreRules, plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, comments, &sb)
	case "text_long":
		return prettyPrintTextLong(parsed, plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, tags, comments, &sb)
	default:
		return "Invalid mode: " + mode
	}
}

func prettyPrintJSON(parsed []*ParsedInfo) string {
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(pretty)
}

func prettyPrintGrepIndex(parsed []*ParsedInfo, ignoreRules []string, sb *strings.Builder) string {
	for _, p := range parsed {
		if len(p.Packages) == 0 {
			continue
		}
		posType, posValue := getPosition(p)
		if shouldIgnorePosition(p, ignoreRules, posType) || len(p.Packages) == 0 {
			continue
		}

		pos_ := formatAndPrint(sb, fmt.Sprintf("%s> %s\n", posType, posValue))

		for _, pkg := range p.Packages {
			pkg_ := formatAndPrint(sb, fmt.Sprintf("package> %s %s", pkg.Package, pos_))
			processStructsForGrepIndex(pkg, pkg_, sb)
			processFunctionsForGrepIndex(pkg, pkg_, sb)
			processVariablesForGrepIndex(pkg, posType, posValue, sb)
			processConstantsForGrepIndex(pkg, pkg_, sb)
			processInterfacesForGrepIndex(pkg, pkg_, sb)
		}
	}
	return sb.String()
}

func prettyPrintLLM(
	parsed []*ParsedInfo,
	ignoreRules []string,
	plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, comments bool,
	sb *strings.Builder,
) string {
	for _, p := range parsed {
		if len(p.Packages) == 0 {
			continue
		}
		printPosition(p, sb)
		for _, pkg := range p.Packages {
			fmt.Fprintf(sb, "Package: %s\n", pkg.Package)
			for _, s := range pkg.Structs {
				if shouldIgnoreStruct(s, pkg.Package, ignoreRules) || !shouldIncludeStruct(s, plainStructs, structsWithMethod) {
					continue
				}
				fmt.Fprintf(sb, "  Struct: %s\n", s.Name)
				if comments && len(s.Docs) > 0 {
					fmt.Fprintf(sb, "    Comment: %s\n", strings.Join(s.Docs, "\n"))
				}
				if fieldsStructsWithMethod || fieldsPlainStructs {
					printFields(s.Fields, ignoreRules, sb, comments)
				}
				if methods && len(s.Methods) > 0 {
					printMethods(s.Methods, ignoreRules, sb)
				}
			}
			if functions && len(pkg.Functions) > 0 {
				printFunctions(pkg.Functions, ignoreRules, sb)
			}
			// interfaces
			for _, i := range pkg.Interfaces {
				fmt.Fprintf(sb, "  Interface: %s\n", i.Name)
				if comments && len(i.Docs) > 0 {
					fmt.Fprintf(sb, "    Comment: %s\n", strings.Join(i.Docs, "\n"))
				}
				for _, m := range i.Methods {
					fmt.Fprintf(sb, "    Method: %s\n", m.Name)
					if comments && len(m.Docs) > 0 {
						fmt.Fprintf(sb, "      Comment: %s\n", strings.Join(m.Docs, "\n"))
					}
					for _, p := range m.Params {
						fmt.Fprintf(sb, "      Param: %s (%s)\n", p.Name, p.Type)
					}
				}
			}
			// variables
			for _, v := range pkg.Variables {
				fmt.Fprintf(sb, "  Variable: %s (%s)\n", v.Name, v.Type)
			}

			// constants
			for _, c := range pkg.Constants {
				fmt.Fprintf(sb, "  Constant: %s\n", c.Name)
			}
		}
	}
	return sb.String()
}

func prettyPrintTextShort(
	parsed []*ParsedInfo,
	ignoreRules []string,
	plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, comments bool,
	sb *strings.Builder,
) string {
	for _, p := range parsed {
		if len(p.Packages) == 0 {
			continue
		}
		printPosition(p, sb)
		for _, pkg := range p.Packages {
			fmt.Fprintln(sb, "Package:", pkg.Package)
			for _, s := range pkg.Structs {
				if shouldIgnoreStruct(s, pkg.Package, ignoreRules) || !shouldIncludeStruct(s, plainStructs, structsWithMethod) {
					continue
				}
				fmt.Fprintln(sb, "  Struct:", s.Name)
				if comments && len(s.Docs) > 0 {
					fmt.Fprintf(sb, "    Comment: %s\n", strings.Join(s.Docs, "\n"))
				}
				if fieldsStructsWithMethod || fieldsPlainStructs {
					for _, f := range s.Fields {
						if shouldIgnoreField(f, ignoreRules) {
							continue
						}
						fmt.Fprintln(sb, "    Field:", f.Name, f.Type)
						if comments && len(f.Docs) > 0 {
							fmt.Fprintf(sb, "      Comment: %s\n", strings.Join(f.Docs, "\n"))
						}
					}
				}
				if methods && len(s.Methods) > 0 {
					for _, m := range s.Methods {
						if shouldIgnoreMethod(m, ignoreRules) {
							continue
						}
						fmt.Fprintln(sb, "    Method:", m.Name)
						if comments && len(m.Docs) > 0 {
							fmt.Fprintf(sb, "      Comment: %s\n", strings.Join(m.Docs, "\n"))
						}
					}
				}
			}
			if functions && len(pkg.Functions) > 0 {
				for _, f := range pkg.Functions {
					if shouldIgnoreFunction(f, ignoreRules) {
						continue
					}
					fmt.Fprintln(sb, "  Function:", f.Name)
					if comments && len(f.Docs) > 0 {
						fmt.Fprintf(sb, "    Comment: %s\n", strings.Join(f.Docs, "\n"))
					}
				}
			}
		}
	}
	return sb.String()
}

func prettyPrintTextLong(
	parsed []*ParsedInfo,
	plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, tags, comments bool,
	sb *strings.Builder,
) string {
	for _, p := range parsed {
		if len(p.Packages) == 0 {
			continue
		}
		printPosition(p, sb)
		for _, pkg := range p.Packages {
			fmt.Fprintf(sb, "Package: %s\n", pkg.Package)
			for _, s := range pkg.Structs {
				if !shouldIncludeStruct(s, plainStructs, structsWithMethod) {
					continue
				}
				fmt.Fprintf(sb, "  Struct: %s\n", s.Name)
				if comments && len(s.Docs) > 0 {
					fmt.Fprintf(sb, "    Comment: %s\n", strings.Join(s.Docs, "\n"))
				}
				if fieldsStructsWithMethod || fieldsPlainStructs {
					for _, f := range s.Fields {
						fmt.Fprintf(sb, "    Field: %s %s\n", f.Name, f.Type)
						if f.Tag != "" && tags {
							fmt.Fprintf(sb, "      Tag: %s\n", f.Tag)
						}
						if f.Comment != "" && comments {
							fmt.Fprintf(sb, "      Comment: %s\n", f.Comment)
						}
					}
				}
				if methods && len(s.Methods) > 0 {
					for _, m := range s.Methods {
						fmt.Fprintf(sb, "    Method: %s\n", m.Name)
						fmt.Fprintf(sb, "      Signature: %s\n", m.Signature)
						if comments && len(m.Docs) > 0 {
							fmt.Fprintf(sb, "      Comment: %s\n", strings.Join(m.Docs, "\n"))
						}
					}
				}
			}
			if functions && len(pkg.Functions) > 0 {
				for _, f := range pkg.Functions {
					fmt.Fprintf(sb, "  Function: %s\n", f.Name)
					fmt.Fprintf(sb, "    Signature: %s\n", f.Signature)
					if comments && len(f.Docs) > 0 {
						fmt.Fprintf(sb, "    Comment: %s\n", strings.Join(f.Docs, "\n"))
					}
				}
			}
		}
	}
	return sb.String()
}

// Helper Functions

func getPosition(p *ParsedInfo) (string, string) {
	if p.File != "" {
		return "file", p.File
	}
	return "directory", p.Directory
}

func shouldIgnorePosition(p *ParsedInfo, ignoreRules []string, posType string) bool {
	if len(ignoreRules) == 0 {
		return false
	}
	posValue := p.Directory
	if posType == "file" {
		posValue = p.File
	}
	for _, ignore := range ignoreRules {
		expression, err := govaluate.NewEvaluableExpression(ignore)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse ignore rule")
			continue
		}
		parameters := map[string]interface{}{posType: posValue}
		result, _ := expression.Evaluate(parameters)
		if result == true {
			return true
		}
	}
	return false
}

func formatAndPrint(sb *strings.Builder, text string) string {
	fmt.Fprint(sb, text)
	return strings.Replace(text, ">", ":", 1)
}

func processStructsForGrepIndex(pkg Package, pkg_ string, sb *strings.Builder) {
	for _, s := range pkg.Structs {
		hasMethods := "has_methods"
		if len(s.Methods) == 0 {
			hasMethods = "no_methods"
		}
		struct_ := formatAndPrint(sb, fmt.Sprintf("struct> %s %s %s", s.Name, hasMethods, pkg_))

		for _, f := range s.Fields {
			formatAndPrint(sb, fmt.Sprintf("field> %s (%s) %s", f.Name, f.Type, struct_))
		}
		for _, m := range s.Methods {
			method_ := formatAndPrint(sb, fmt.Sprintf("method> %s %s", m.Name, struct_))
			paramsString := processParamsForGrepIndex(m.Params, method_, sb)
			for _, r := range m.Returns {
				formatAndPrint(sb, fmt.Sprintf("return> %s %s %s", r.Type, paramsString, method_))
			}
		}
	}
}

func processParamsForGrepIndex(params []Param, method_ string, sb *strings.Builder) string {
	var paramsSlice []string
	for _, p := range params {
		formatAndPrint(sb, fmt.Sprintf("param> %s (%s) %s", p.Name, p.Type, method_))
		paramsSlice = append(paramsSlice, fmt.Sprintf("param %s (%s)", p.Name, p.Type))
	}
	if len(paramsSlice) == 0 {
		return "no_params"
	}
	return strings.Join(paramsSlice, " ")
}

func processFunctionsForGrepIndex(pkg Package, pkg_ string, sb *strings.Builder) {
	for _, f := range pkg.Functions {
		function_ := formatAndPrint(sb, fmt.Sprintf("function> %s %s", f.Name, pkg_))
		for _, p := range f.Params {
			formatAndPrint(sb, fmt.Sprintf("param> %s (%s) %s", p.Name, p.Type, function_))
		}
	}
}

func processVariablesForGrepIndex(pkg Package, posType, posValue string, sb *strings.Builder) {
	for _, v := range pkg.Variables {
		formatAndPrint(sb, fmt.Sprintf("variable> %s (%s) package %s %s:%s", v.Name, v.Type, pkg.Package, posType, posValue))
	}
}

func processConstantsForGrepIndex(pkg Package, pkg_ string, sb *strings.Builder) {
	for _, c := range pkg.Constants {
		formatAndPrint(sb, fmt.Sprintf("constant> %s %s", c.Name, pkg_))
	}
}

func processInterfacesForGrepIndex(pkg Package, pkg_ string, sb *strings.Builder) {
	for _, i := range pkg.Interfaces {
		interface_ := formatAndPrint(sb, fmt.Sprintf("interface> %s %s", i.Name, pkg_))
		for _, m := range i.Methods {
			method_ := formatAndPrint(sb, fmt.Sprintf("method> %s %s", m.Name, interface_))
			for _, p := range m.Params {
				formatAndPrint(sb, fmt.Sprintf("param> %s (%s) %s", p.Name, p.Type, method_))
			}
		}
	}
}

func printPosition(p *ParsedInfo, sb *strings.Builder) {
	if p.Directory != "" {
		fmt.Fprintf(sb, "Directory: %s\n", p.Directory)
	} else if p.File != "" {
		fmt.Fprintf(sb, "File: %s\n", p.File)
	}
}

func shouldIgnoreStruct(s Struct, pkgName string, ignoreRules []string) bool {
	if len(ignoreRules) == 0 {
		return false
	}
	structParams := map[string]interface{}{
		"package":            pkgName,
		"struct_name":        s.Name,
		"struct_len_fields":  len(s.Fields),
		"struct_len_methods": len(s.Methods),
	}
	for _, ignore := range ignoreRules {
		expression, err := govaluate.NewEvaluableExpression(ignore)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse ignore rule")
			continue
		}
		result, _ := expression.Evaluate(structParams)
		if result == true {
			log.Trace().Str("rule", ignore).Msgf("Ignoring struct %s", s.Name)
			return true
		}
	}
	return false
}

func shouldIncludeStruct(s Struct, plainStructs, structsWithMethod bool) bool {
	if !plainStructs && len(s.Methods) == 0 {
		return false
	}
	if !structsWithMethod && len(s.Methods) > 0 {
		return false
	}
	return true
}

func printFields(fields []Field, ignoreRules []string, sb *strings.Builder, comments bool) {
	var fieldNames []string
	for _, f := range fields {
		if shouldIgnoreField(f, ignoreRules) {
			continue
		}
		fieldNames = append(fieldNames, fmt.Sprintf("%s (%s)", f.Name, f.Type))
	}
	if len(fieldNames) > 0 {
		fmt.Fprintf(sb, "    %s\n", strings.Join(fieldNames, ", "))
	}
}

func shouldIgnoreField(f Field, ignoreRules []string) bool {
	if len(ignoreRules) == 0 {
		return false
	}
	fieldParams := map[string]interface{}{
		"field_name": f.Name,
		"field_type": f.Type,
	}
	for _, ignore := range ignoreRules {
		expression, err := govaluate.NewEvaluableExpression(ignore)
		if err != nil {
			log.Warn().Str("rule", ignore).Err(err).Msg("Failed to parse ignore rule")
			continue
		}
		result, _ := expression.Evaluate(fieldParams)
		if result == true {
			log.Trace().Str("rule", ignore).Msgf("Ignoring field %s", f.Name)
			return true
		}
	}
	return false
}

func printMethods(methods []Method, ignoreRules []string, sb *strings.Builder) {
	var methodNamesAndSig []string
	for _, m := range methods {
		if shouldIgnoreMethod(m, ignoreRules) {
			continue
		}
		methodNamesAndSig = append(methodNamesAndSig, m.Signature)
	}
	if len(methodNamesAndSig) > 0 {
		fmt.Fprintf(sb, "    Methods: %s\n", strings.Join(methodNamesAndSig, ", "))
	}
}

func shouldIgnoreMethod(m Method, ignoreRules []string) bool {
	if len(ignoreRules) == 0 {
		return false
	}
	methodParams := map[string]interface{}{
		"method_name":        m.Name,
		"method_len_params":  len(m.Params),
		"method_len_returns": len(m.Returns),
	}
	for _, ignore := range ignoreRules {
		expression, err := govaluate.NewEvaluableExpression(ignore)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse ignore rule")
			continue
		}
		result, _ := expression.Evaluate(methodParams)
		if result == true {
			log.Trace().Str("rule", ignore).Msgf("Ignoring method %s", m.Name)
			return true
		}
	}
	return false
}

func printFunctions(functions []Function, ignoreRules []string, sb *strings.Builder) {
	var funcNames []string
	for _, f := range functions {
		if shouldIgnoreFunction(f, ignoreRules) {
			continue
		}
		funcNames = append(funcNames, f.Name)
	}
	if len(funcNames) > 0 {
		fmt.Fprintf(sb, "  Functions: %s\n", strings.Join(funcNames, ", "))
	}
}

func shouldIgnoreFunction(f Function, ignoreRules []string) bool {
	if len(ignoreRules) == 0 {
		return false
	}
	funcParams := map[string]interface{}{
		"function_name":        f.Name,
		"function_len_params":  len(f.Params),
		"function_len_returns": len(f.Returns),
	}
	for _, ignore := range ignoreRules {
		expression, err := govaluate.NewEvaluableExpression(ignore)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse ignore rule")
			continue
		}
		result, _ := expression.Evaluate(funcParams)
		if result == true {
			log.Trace().Str("rule", ignore).Msgf("Ignoring function %s", f.Name)
			return true
		}
	}
	return false
}
