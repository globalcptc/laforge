//go:build ignore
// +build ignore

package main

import (
	"log"

	"entgo.io/contrib/entgql"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/hedwigz/entviz"
)

func main() {
	ex, err := entgql.NewExtension()
	if err != nil {
		log.Fatalf("creating entgql extension: %v", err)
	}
	opts := []entc.Option{
		entc.TemplateFiles("template/ent.tmpl"),
		entc.Extensions(entviz.Extension{}, ex),
	}
	generr := entc.Generate("./schema", &gen.Config{}, opts...)
	if generr != nil {
		log.Fatalf("running ent codegen: %v", generr)
	}
}
