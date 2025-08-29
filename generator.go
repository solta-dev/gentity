package main

import (
	"bytes"
	"embed"
	_ "embed"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/exp/maps"
)

//go:embed *.go.tmpl
var templatesFS embed.FS
var templates *template.Template

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

	templates = template.New("file.go.tmpl")
	templates.Funcs(tmplFuncs)
	if _, err := templates.ParseFS(templatesFS, "*.go.tmpl"); err != nil {
		log.Fatalf("ParseFS: %v", err)
	}

	templates.Funcs(tmplFuncs)
}

func generate(packageName string, entities []entity) string {
	newFileName := "gentity.gen.go"

	var imports map[string]struct{} = make(map[string]struct{})
	for _, entity := range entities {
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

		if len(entity.JsonFields) > 0 {
			imports["encoding/json"] = struct{}{}
		}
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].GoName < entities[j].GoName
	})

	var buf bytes.Buffer
	if err := templates.Execute(&buf, struct {
		PackageName string
		Entities    []entity
		Imports     []string
	}{packageName, entities, maps.Keys(imports)}); err != nil {
		log.Fatalf("Execute template: %v", err)
	}

	outFile, err := os.Create(newFileName)
	if err != nil {
		log.Fatalf("Create file: %v", err)
	}
	defer outFile.Close()
	if _, err := outFile.WriteString(buf.String()); err != nil {
		log.Fatalf("Failed to write generated file %s: %v", newFileName, err)
	}

	return newFileName
}
