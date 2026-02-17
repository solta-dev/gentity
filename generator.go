package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
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

func generate(packageName string, entities []entity) (newFileName string, err error) {
	newFileName = "gentity.gen.go"

	imports := make(map[string]string)
	for _, entity := range entities {
		for _, field := range entity.Fields {
			importsByField(&entity, field, imports)
		}

		if len(entity.JsonFields) > 0 {
			imports["encoding/json"] = "encoding/json"
		}
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].GoName < entities[j].GoName
	})

	var buf bytes.Buffer
	if err = templates.Execute(&buf, struct {
		PackageName string
		Entities    []entity
		Imports     map[string]string
	}{packageName, entities, imports}); err != nil {
		err = fmt.Errorf("execute template failed: %v", err)
		return
	}

	var outFile *os.File
	outFile, err = os.Create(newFileName)
	if err != nil {
		err = fmt.Errorf("create file: %v", err)
		return
	}
	defer outFile.Close()
	if _, err = outFile.WriteString(buf.String()); err != nil {
		err = fmt.Errorf("failed to write generated file %s: %v", newFileName, err)
		return
	}

	return
}

func importsByField(entity *entity, field *field, imports map[string]string) {
	// Import field type need only if it used arguments of methods.
	// This is one case: getters.
	if len(field.InIndexes) == 0 && !field.InPrimaryKey {
		return
	}

	t := strings.Split(field.GoType, ".")
	if len(t) == 1 {
		return
	}

	pkgAlias := t[0]
	if len(pkgAlias) > 2 && pkgAlias[:2] == "[]" {
		pkgAlias = pkgAlias[2:]
	}
	if len(pkgAlias) > 1 && pkgAlias[0] == '*' {
		pkgAlias = pkgAlias[1:]
	}

	if pkgAlias == "pgtype" {
		imports["pgtype"] = "github.com/jackc/pgx/v5/pgtype"
	} else if imp, ok := entity.Imports[pkgAlias]; ok {
		imports[pkgAlias] = imp
	} else {
		imports[pkgAlias] = pkgAlias
	}
}
