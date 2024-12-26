package main

import (
	"github.com/wricardo/code-surgeon/examples/structparser2"
	"github.com/wricardo/code-surgeon/neo4j2"
)

func main() {
	x, _ := neo4j2.NewStackToGraph("bolt://localhost:7687", "neo4j", "neo4jneo4j")
	x.SetupGlobal()
	p := structparser2.Person{Name: "Sasha"}
	c := structparser2.Controller{}
	c.GetPerson("Sasha")
	structparser2.GlobalFuncWithParam(p)
}
