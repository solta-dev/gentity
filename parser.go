package main

import (
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

type tag struct {
	byNumber []string
	byName   map[string]string
}

func newTag(text string, name string) *tag {
	tags := make(map[string]tag)

	const (
		outer uint8 = iota
		initial
		tagString
		optionName
		optionValue
	)

	var (
		state          uint8 = outer
		start          int
		tagName        string
		optName        string
		curTag         *tag
		outerDelimiter byte
	)
	for i := 0; i < len(text); i++ {
		//fmt.Printf("i=%d c=%c state=%d\n", i, text[i], state)
		switch state {
		case outer:
			outerDelimiter = text[i]
			state = initial
			start = i + 1
		case initial:
			switch text[i] {
			case byte(':'), outerDelimiter:
				tagName = text[start:i]
				state = tagString
				curTag = &tag{byName: make(map[string]string, 1), byNumber: make([]string, 0, 1)}
				tags[tagName] = *curTag
			case byte(' '):
				start = i + 1
			}
		case tagString:
			switch text[i] {
			case byte('"'):
				state = optionName
				start = i + 1
				optName = ""
			case outerDelimiter:
			default:
				log.Fatalf("Tag delimiter %c doesn't supported", text[i])
			}
		case optionName:
			switch text[i] {
			case byte('='):
				optName = text[start:i]
				state = optionValue
				start = i + 1
			case byte(' '), byte('"'):
				optName = text[start:i]
				curTag.byNumber = append(tags[tagName].byNumber, optName)
				curTag.byName[optName] = ""
				if text[i] == byte('"') {
					state = initial
				}
				start = i + 1
			}
		case optionValue:
			if text[i] == byte(' ') || text[i] == byte('"') {
				optValue := text[start:i]
				curTag.byName[optName] = optValue
				if text[i] == byte(' ') {
					state = optionName
				} else if text[i] == byte('"') {
					state = initial
				}
				start = i + 1
			}
		}
	}

	tag, ok := tags[name]
	if ok {
		return &tag
	} else {
		return nil
	}
}

type structInfo struct {
	typeSpec     *ast.TypeSpec
	structType   *ast.StructType
	isForGentity bool
}

type structsMap map[string]structInfo

func procFieldTag(f *ast.Field, ent *entity, fld *field) {
	if f.Tag != nil {
		fmt.Println(ent.GoName, "field", fld.GoName, "tag", f.Tag.Value)
		tag := newTag(f.Tag.Value, "gentity")
		if tag != nil {
			if index, ok := tag.byName["index"]; ok {
				if _, ok := ent.NonUniqIndexes[index]; !ok {
					ent.NonUniqIndexes[index] = []*field{fld}
				} else {
					ent.NonUniqIndexes[index] = append(ent.NonUniqIndexes[index], fld)
				}
			}
			if unique, ok := tag.byName["unique"]; ok {
				fmt.Println(ent.GoName, "uniq", unique, "field", fld.GoName)
				if _, ok := ent.UniqIndexes[unique]; !ok {
					ent.UniqIndexes[unique] = []*field{fld}
				} else {
					ent.UniqIndexes[unique] = append(ent.UniqIndexes[unique], fld)
				}
			}
			if _, ok := tag.byName["autoincrement"]; ok {
				ent.AutoIncrementField = fld
			}
		}
	} else {
		fmt.Println(ent.GoName, "field", fld.GoName, "no tag")
	}
}

func (sm structsMap) procType(f *ast.Field, e ast.Expr, ent *entity) {
	switch e.(type) {
	case *ast.Ident:
		id := e.(*ast.Ident)

		fmt.Println(ent.GoName, "fields:", f.Names, "ident.name:", id.Name)
		if len(f.Names) == 0 {
			if _, ok := sm[id.Name]; !ok {
				log.Fatalf("Embedded structure %s wasn't found in package", id.Name)
			}

			subt := sm.procStruct(id.Name)

			subt.Fields[0].OpeningEmbed = append(subt.Fields[0].OpeningEmbed, id.Name)
			subt.Fields[len(subt.Fields)-1].ClosingEmbed = append(subt.Fields[0].ClosingEmbed, id.Name)
			for i := range subt.Fields {
				subt.Fields[i].EmbedLevel++
				subt.Fields[i].Num = len(ent.Fields) + i
			}
			for name, uniq := range subt.UniqIndexes {
				ent.UniqIndexes[name] = uniq
			}
			for name, index := range subt.NonUniqIndexes {
				ent.NonUniqIndexes[name] = index
			}
			ent.Fields = append(ent.Fields, subt.Fields...)
		} else {
			fld := newField()
			fld.GoName = f.Names[0].Name
			fld.SQLName = camelCaseToSnakeCase(f.Names[0].Name)
			fld.GoType = e.(*ast.Ident).Name
			fld.Num = len(ent.Fields)
			procFieldTag(f, ent, fld)
			ent.Fields = append(ent.Fields, *fld)
		}
	case *ast.SelectorExpr:
		fld := newField()
		fld.GoName = f.Names[0].Name
		fld.SQLName = camelCaseToSnakeCase(f.Names[0].Name)
		fld.GoType = e.(*ast.SelectorExpr).Sel.Name
		fld.Num = len(ent.Fields)
		if expX, ok := e.(*ast.SelectorExpr).X.(*ast.Ident); ok {
			fld.GoType = expX.Name + "." + fld.GoType
		}
		procFieldTag(f, ent, fld)
		ent.Fields = append(ent.Fields, *fld)
	case *ast.StarExpr:
		se := e.(*ast.StarExpr)
		sm.procType(f, se.X, ent)
		ent.Fields[len(ent.Fields)-1].IsRef = true
	case *ast.ArrayType:
		at := e.(*ast.ArrayType)
		sm.procType(f, at.Elt, ent)
		ent.Fields[len(ent.Fields)-1].IsArray = true
	default:
		id := e.(*ast.Ident)
		log.Fatalf("Unknown type of field %s.%s type: %+v", ent.GoName, id.Name, e)
	}
}

func (sm structsMap) procStruct(name string) (e *entity) {

	e = newEntity()

	if _, ok := sm[name]; !ok {
		log.Fatalf("Struct type %s not found in package", name)
	}

	e.GoName = sm[name].typeSpec.Name.Name
	e.SQLName = camelCaseToSnakeCase(mkPlural(e.GoName))

	for _, f := range sm[name].structType.Fields.List {
		sm.procType(f, f.Type, e)
	}

	for i, f := range e.Fields {

		//e.Fields[i].Num = i

		if f.IsArray {
			e.Fields[i].GoType = "[]" + f.GoType
		}

		if f.IsRef {
			e.Fields[i].GoType = "*" + f.GoType
		}
	}

	for name, fields := range e.UniqIndexes {
		if name == "primary" || (e.PrimaryIndex != "" && len(e.UniqIndexes[e.PrimaryIndex]) > len(fields)) || e.PrimaryIndex == "" {
			e.PrimaryIndex = name
		}
		for _, f := range fields {
			fmt.Println(e.GoName, name, "uniq", f.GoName)
		}
	}
	if e.PrimaryIndex != "" {
		e.FieldsExcludePrimaryKey = make([]field, 0, len(e.Fields)-len(e.UniqIndexes[e.PrimaryIndex]))
		for _, f := range e.UniqIndexes[e.PrimaryIndex] {
			e.Fields[f.Num].InPrimaryKey = true
		}
		for _, f := range e.Fields {
			if !f.InPrimaryKey {
				e.FieldsExcludePrimaryKey = append(e.FieldsExcludePrimaryKey, f)
			}
		}
	} else {
		e.FieldsExcludePrimaryKey = e.Fields
	}

	for name, fields := range e.UniqIndexes {
		for _, f := range fields {
			f.InIndexes = append(f.InIndexes, name)
		}
	}
	for name, fields := range e.NonUniqIndexes {
		for _, f := range fields {
			f.InIndexes = append(f.InIndexes, name)
		}
	}

	if e.AutoIncrementField == nil {
		e.FieldsExcludeAutoIncrement = e.Fields
	} else {
		e.FieldsExcludeAutoIncrement = make([]field, 0, len(e.Fields)-1)
		for _, f := range e.Fields {
			if e.AutoIncrementField.GoName != f.GoName {
				e.FieldsExcludeAutoIncrement = append(e.FieldsExcludeAutoIncrement, f)
			}
		}
	}

	return
}

func parse() (packageName string, entities []entity) {

	path := os.Getenv("GOFILE")
	if path == "" {
		log.Fatal("GOFILE must be set")
	}

	astPkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Dir(path), nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("parse dir: %v", err)
	}
	if len(astPkgs) != 1 {
		log.Fatalf("Not one package found")
	}

	var files []*ast.File
	for _, p := range astPkgs {
		files = append(files, maps.Values(p.Files)...)
		packageName = p.Name
	}

	structs := structsMap(make(map[string]structInfo))

	inspector.New(files).Nodes([]ast.Node{&ast.GenDecl{}}, func(node ast.Node, push bool) (proceed bool) {
		genDecl := node.(*ast.GenDecl)

		si := structInfo{}
		var ok bool

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

		structs[si.typeSpec.Name.Name] = si

		return false
	})

	for name, si := range structs {
		if !si.isForGentity {
			continue
		}
		//log.Println("proc struct", name)
		entity := structs.procStruct(name)
		//log.Printf("entity %s: %+v", name, entity)

		entities = append(entities, *entity)
	}

	return
}
