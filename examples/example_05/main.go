package main

import (
	"fmt"

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

	fragments := []codesurgeon.CodeFragment{}

	for _, s := range pkg.Functions {
		if len(s.Docs) > 0 && len(s.Docs[0]) > 0 {
			continue
		}
		fmt.Println(s.Signature + " needs doc")
		content := "// " + s.Name + " my automatic comment\nfunc " + s.Signature + s.Body
		fmt.Println(content)
		fragments = append(fragments, codesurgeon.CodeFragment{
			Content:   content,
			Overwrite: true,
		})

	}

	fmap := map[string][]codesurgeon.CodeFragment{
		"other.go": fragments,
	}

	err = codesurgeon.InsertCodeFragments(fmap)
	if err != nil {
		fmt.Println(err)
		return
	}
}
