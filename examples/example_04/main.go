package main

import (
	"fmt"

	codesurgeon "github.com/wricardo/code-surgeon"
)

type INode interface {
	GetID() string
	SetID(id string)
	GetLabel() string
	GetProps() map[string]any
	SetProps(props map[string]any)
}

func main() {
	tmp, err := codesurgeon.ParseFile("other.go")
	if err != nil {
		fmt.Println(err)
		return
	}
	pkg := tmp.Packages[0]
	fragment := make([]codesurgeon.CodeFragment, 0)
	for _, s := range pkg.Structs {
		fragment = append(fragment, codesurgeon.CodeFragment{
			Content: codesurgeon.RenderTemplateNoError(`
			func (n {{.Name}}) GetID() string {
				return n.Id
			}

			func (n {{.Name}}) SetID(id string) {
				n.Id = id
			}	

			func (n {{.Name}}) GetLabel() string {
				return "{{.Name}}"
			}

			func (n {{.Name}}) GetProps() map[string]any {
				return map[string]any{
				}
			}

			func (n {{.Name}}) SetProps(props map[string]any) {
				if val, ok := props["SomeField"]; ok {
					n.SomeField = val.(string)
				}
			}




			`, s),

			Overwrite: false,
		})
	}

	fragments := map[string][]codesurgeon.CodeFragment{
		"other.go": fragment,
	}

	codesurgeon.InsertCodeFragments(fragments)
}
