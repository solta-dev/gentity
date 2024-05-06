package main

import (
	"bytes"
	_ "embed"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/exp/maps"
)

var fileTmpl, entityTmpl *template.Template

//go:embed file.go.tmpl
var fileTmplContent []byte

//go:embed entity.go.tmpl
var entityTmplContent []byte

func init() {
	tmplFuncs := template.FuncMap{
		"join": func(sep string, s []string) string { return strings.Join(s, sep) },
		"add": func(x ...int) (y int) {
			for _, v := range x {
				y += v
			}
			return
		},
		"n_tabs":                   func(n int) string { return strings.Repeat("\t", n) },
		"snake_case_to_camel_case": snakeCaseToCamelCase,
		"exists":                   func(m map[string]any, key string) bool { _, ok := m[key]; return ok },
	}

	fileTmpl = template.Must(template.New("file").Funcs(tmplFuncs).Parse(string(fileTmplContent)))
	entityTmpl = template.Must(template.New("entity").Funcs(tmplFuncs).Parse(string(entityTmplContent)))
}

func generate(packageName string, entities []entity) {
	newFileName := "gentity.gen.go"

	var entitiesCode []string
	var imports map[string]struct{} = make(map[string]struct{})
	for _, entity := range entities {

		var buf bytes.Buffer
		if err := entityTmpl.Execute(&buf, entity); err != nil {
			log.Fatalf("Execute template: %v", err)
		}

		entitiesCode = append(entitiesCode, buf.String())

		for _, field := range entity.Fields {
			// Import field type need only if it used arguments of methods.
			// This is one case: getters.
			if len(field.InIndexes) == 0 {
				continue
			}

			t := strings.Split(field.GoType, ".")
			if len(t) == 1 {
				continue
			}

			if t[0] == "pgtype" {
				imports["github.com/jackc/pgx/v5/pgtype"] = struct{}{}
			} else {
				imports[t[0]] = struct{}{}
			}
		}
	}
	sort.Strings(entitiesCode)

	var buf bytes.Buffer
	if err := fileTmpl.Execute(&buf, struct {
		PackageName string
		Entities    []string
		Imports     []string
	}{packageName, entitiesCode, maps.Keys(imports)}); err != nil {
		log.Fatalf("Execute template: %v", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), newFileName, buf.Bytes(), parser.ParseComments); err != nil {
		log.Fatalf("Parse template failed: %v; template below: %s", err, buf.String())
	}

	outFile, err := os.Create(newFileName)
	if err != nil {
		log.Fatalf("Create file: %v", err)
	}
	defer outFile.Close()
	if _, err := outFile.WriteString(buf.String()); err != nil {
		log.Fatalf("Failed to write generated file %s: %v", newFileName, err)
	}
}
