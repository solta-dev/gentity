package main

import (
	"bytes"
	_ "embed"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/template"
)

var fileTmpl, entityTmpl *template.Template

//go:embed file.go.tmpl
var fileTmplContent []byte

//go:embed entity.go.tmpl
var entityTmplContent []byte

func init() {
	tmplFuncs := template.FuncMap{
		"join":                     func(sep string, s []string) string { return strings.Join(s, sep) },
		"is_last":                  func(x int, a interface{}) bool { return x == reflect.ValueOf(a).Len()-1 },
		"add":                      func(x int, y int) int { return x + y },
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
	for _, entity := range entities {

		var buf bytes.Buffer
		if err := entityTmpl.Execute(&buf, entity); err != nil {
			log.Fatalf("Execute template: %v", err)
		}

		entitiesCode = append(entitiesCode, buf.String())
	}
	sort.Strings(entitiesCode)

	var buf bytes.Buffer
	if err := fileTmpl.Execute(&buf, struct {
		PackageName string
		Entities    []string
	}{packageName, entitiesCode}); err != nil {
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
