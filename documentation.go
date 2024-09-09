package codesurgeon

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// UpsertDocumentationToFunction uses Comby to upsert documentation in a function. Replace the existing documentation if it exists.
// It returns true if the documentation was updated, false otherwise.
func UpsertDocumentationToFunction(filePath, receiver, functionName, documentation string) (bool, error) {
	if filePath == "" {
		return false, fmt.Errorf("file path is empty")
	}
	if functionName == "" {
		return false, fmt.Errorf("function name is empty")
	}
	type matchRewrite struct {
		Match    string
		Rewrite  string
		Rule     string
		Continue bool
	}

	FormatCodeAndFixImports(filePath)

	mrs := []matchRewrite{}

	if !strings.HasPrefix(documentation, "//") {
		return false, fmt.Errorf("documentation should start with //")
	}

	if receiver != "" {
		// Patterns for methods associated with a struct
		mrs = append(mrs,
			// // Replace existing documentation above a method
			matchRewrite{
				Match:    fmt.Sprintf(":[comments~(//[^\n]*\n)*]func (:[receiver] %s) %s(:[args]) :[rest]", receiver, functionName),
				Rewrite:  fmt.Sprintf("func (:[receiver] %s) %s(:[args]) :[rest]", receiver, functionName),
				Rule:     fmt.Sprintf("where rewrite :[comments] { \"//:[comment]\" -> \"%s\" }", documentation),
				Continue: true,
			},
			// // Replace existing documentation above a method with Pointer receiver
			matchRewrite{
				Match:    fmt.Sprintf(":[comments~(//[^\n]*\n)*]func (:[receiver] *%s) %s(:[args]) :[rest]", receiver, functionName),
				Rewrite:  fmt.Sprintf("func (:[receiver] *%s) %s(:[args]) :[rest]", receiver, functionName),
				Rule:     fmt.Sprintf("where rewrite :[comments] { \"//:[comment]\" -> \"%s\" }", documentation),
				Continue: true,
			},
			// Add new documentation to a method without documentation
			matchRewrite{
				Match:   fmt.Sprintf(":[a~\n]:[b~\n]func (:[receiver]%s) %s(:[c])", receiver, functionName),
				Rewrite: fmt.Sprintf(":[a]%s\nfunc (:[receiver]%s) %s(:[c])", documentation, receiver, functionName),
			},
			// Add new documentation to a method without documentation with Pointer receiver
			matchRewrite{
				Match:   fmt.Sprintf(":[a~\n]:[b~\n]func (:[receiver]*%s) %s(:[c])", receiver, functionName),
				Rewrite: fmt.Sprintf(":[a]%s\nfunc (:[receiver]*%s) %s(:[c])", documentation, receiver, functionName),
			},
		)
	} else {
		// Updated Match, Rewrite, and Rule
		mrs = append(mrs,
			matchRewrite{
				Match:    fmt.Sprintf(":[comments~(//[^\n]*\n)*]func %s(:[args]) :[rest]", functionName),
				Rewrite:  fmt.Sprintf("func %s(:[args]) :[rest]", functionName),
				Rule:     fmt.Sprintf("where rewrite :[comments] { \"//:[comment]\" -> \"// %s\" }", documentation),
				Continue: true,
			},
			matchRewrite{
				Match:   fmt.Sprintf(":[a~\n]:[b~\n]func %s(:[c])", functionName),
				Rewrite: fmt.Sprintf(":[a]%s:[b]func %s(:[c])", documentation, functionName),
			},
		)
	}

	modified := false
	for _, mr := range mrs {
		args := []string{mr.Match, mr.Rewrite, filePath, "-json-lines", "-match-newline-at-toplevel", "-matcher", ".go"}
		if mr.Rule != "" {
			args = append(args, "-rule", mr.Rule)
		}
		cmd := exec.Command("comby", args...)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		if err != nil {
			return modified, fmt.Errorf("failed to run comby: %w, output: %s", err, string(output))
		}

		if outputStr != "" {
			expectedOutput := &struct {
				RewrittenSource string `json:"rewritten_source"`
			}{}
			if err := json.Unmarshal(output, expectedOutput); err != nil {
				return modified, fmt.Errorf("failed to unmarshal comby output: %w", err)
			}
			if expectedOutput.RewrittenSource == "" {
				continue
			}
			err = writeFile(filePath, expectedOutput.RewrittenSource)
			if err != nil {
				return modified, fmt.Errorf("failed to write file: %w", err)
			}
			FormatCodeAndFixImports(filePath)
			modified = true
			if !mr.Continue {
				return modified, nil
			}
		}
	}

	return modified, nil
}
