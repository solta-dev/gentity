package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/exp/maps"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/go/ast/inspector"
)

func camelCaseToSnakeCase(camel string) (snake string) {
	re := regexp.MustCompile("([a-z])([A-Z]+)")
	snake = re.ReplaceAllString(camel, "${1}_${2}")
	snake = strings.ToLower(snake)
	return
}

func snakeCaseToCamelCase(snake string, firstLetterUpperCase bool) (camel string) {
	titler := cases.Title(language.AmericanEnglish)

	words := strings.Split(snake, "_")

	if firstLetterUpperCase {
		camel = titler.String(words[0])
	} else {
		camel = words[0]
	}

	for _, word := range words[1:] {
		camel += titler.String(word)
	}
	return
}

func mkPlural(singular string) (plural string) {
	re := regexp.MustCompile("y$")
	plural = re.ReplaceAllString(singular, "ie")
	plural += "s"
	return
}

func procFieldTag(f *ast.Field, ent *entity, fld *field) error {
	if f.Tag == nil {
		log.Println(ent.GoName, "field", fld.GoName, "no tag")
		return nil
	}
	log.Println(ent.GoName, "field", fld.GoName, "tag", f.Tag.Value)
	tp := tagParser{Text: f.Tag.Value}
	if err := tp.parse(); err != nil {
		return err
	}
	tag, tagExists := tp.Result["gentity"]
	if !tagExists {
		return nil
	}
	if index, ok := tag["index"]; ok {
		if _, ok := ent.NonUniqIndexes[index]; !ok {
			ent.NonUniqIndexes[index] = []*field{fld}
		} else {
			ent.NonUniqIndexes[index] = append(ent.NonUniqIndexes[index], fld)
		}
	}
	if unique, ok := tag["unique"]; ok {
		log.Println(ent.GoName, "uniq", unique, "field", fld.GoName)
		if _, ok := ent.UniqIndexes[unique]; !ok {
			ent.UniqIndexes[unique] = []*field{fld}
		} else {
			ent.UniqIndexes[unique] = append(ent.UniqIndexes[unique], fld)
		}
	}
	if _, ok := tag["autoincrement"]; ok {
		ent.AutoIncrementField = fld
	}
	return nil
}

func parse() (packageName string, entities []entity, err error) {
	path := os.Getenv("GOFILE")
	if path == "" {
		return "", nil, errors.New("GOFILE must be set")
	}

	astPkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Dir(path), nil, parser.ParseComments)
	if err != nil {
		return "", nil, fmt.Errorf("parse dir: %v", err)
	}
	if len(astPkgs) != 1 {
		return "", nil, errors.New("not one package found")
	}

	var files []*ast.File
	for _, p := range astPkgs {
		files = append(files, maps.Values(p.Files)...)
		packageName = p.Name
	}

	imports := make(map[string]string)
	structs := structsMap(make(map[string]structInfo))

	inspector.New(files).Nodes([]ast.Node{&ast.GenDecl{}}, func(node ast.Node, _ bool) (proceed bool) { return procNode(node, structs, imports) })

	for name, si := range structs {
		if !si.isForGentity {
			continue
		}
		var entity *entity
		entity, err = structs.procStruct(name, imports)
		if err != nil {
			return
		}
		entities = append(entities, *entity)
	}

	return
}

func procNode(node ast.Node, sm structsMap, imports map[string]string) (proceed bool) {
	genDecl, ok := node.(*ast.GenDecl)
	if !ok {
		panic("unexpected node type")
	}

	si := structInfo{}

	for _, spec := range genDecl.Specs {
		if is, ok := spec.(*ast.ImportSpec); ok {
			var alias string
			if is.Name == nil {
				pathParts := strings.Split(is.Path.Value, "/")
				alias = pathParts[len(pathParts)-1]
			} else {
				alias = is.Name.Name
			}
			imports[strings.Trim(alias, "\"")] = strings.Trim(is.Path.Value, "\"")
		}
	}

	si.typeSpec, ok = genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return false
	}

	si.structType, ok = si.typeSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	if genDecl.Doc != nil {
		for _, comment := range genDecl.Doc.List {
			if comment.Text == "// gentity" {
				si.isForGentity = true
			}
		}
	}

	sm[si.typeSpec.Name.Name] = si

	return false
}
