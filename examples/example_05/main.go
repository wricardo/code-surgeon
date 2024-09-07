package main

import (
	"fmt"
	"strings"

	codesurgeon "github.com/wricardo/code-surgeon"
	structparser "github.com/wricardo/structparser"
)

func main() {
	tmp, err := structparser.ParseFile("other.go")
	if err != nil {
		fmt.Println(err)
		return
	}
	pkg := tmp.Packages[0]

	for _, s := range pkg.Functions {
		if len(s.Docs) > 0 && len(s.Docs[0]) > 0 {
			continue
		}
		content := "" + s.Name + " my automatic comment"
		modified, err := codesurgeon.UpsertDocumentationToFunction("other.go", "", s.Name, content)
		if err != nil {
			fmt.Println(err)
			return
		}
		if modified {
			fmt.Println("Documentation added to function", s.Name)
		}

	}

	for _, s := range pkg.Structs {
		for _, m := range s.Methods {
			content := "" + m.Name + " my automatic comment for method for struct " + s.Name
			m.Receiver = strings.Replace(m.Receiver, "*", "", -1)
			modified, err := codesurgeon.UpsertDocumentationToFunction("other.go", m.Receiver, m.Name, content)
			if err != nil {
				fmt.Println(err)
				return
			}
			if modified {
				fmt.Println("Documentation added to method", m.Name)
			}
		}

	}

}
