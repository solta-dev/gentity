package main

import (
	"fmt"
	"go/ast"
	"slices"
)

type structInfo struct {
	typeSpec     *ast.TypeSpec
	structType   *ast.StructType
	isForGentity bool
}

type structsMap map[string]structInfo

func (sm structsMap) procType(f *ast.Field, e ast.Expr, ent *entity, imports map[string]string) (err error) {
	switch e := e.(type) {
	case *ast.Ident:
		err = sm.procTypeIdent(f, e, ent, imports)
	case *ast.SelectorExpr:
		err = sm.procTypeSelectorExpr(f, e, ent, imports)
	case *ast.StarExpr:
		err = sm.procType(f, e.X, ent, imports)
		ent.Fields[len(ent.Fields)-1].IsRef = true
	case *ast.ArrayType:
		err = sm.procType(f, e.Elt, ent, imports)
		ent.Fields[len(ent.Fields)-1].IsArray = true
	default:
		if id, ok := e.(*ast.Ident); ok {
			err = fmt.Errorf("unknown type of field %s.%s type: %+v", ent.GoName, id.Name, e)
		} else {
			err = fmt.Errorf("unknown type of field %s type: %+v", ent.GoName, e)
		}
	}
	for i, fld := range ent.Fields {
		if fld.IsArray {
			ent.Fields[i].GoType = "[]" + fld.GoType
		}
		if fld.IsRef {
			ent.Fields[i].GoType = "*" + fld.GoType
		}
	}
	return
}
func (sm structsMap) procTypeIdent(f *ast.Field, e *ast.Ident, ent *entity, imports map[string]string) error {
	if len(f.Names) == 0 {
		return sm.procTypeEmbed(e, ent, imports)
	}

	fld := newField()
	fld.GoName = f.Names[0].Name
	fld.SQLName = camelCaseToSnakeCase(f.Names[0].Name)
	fld.GoType = e.Name
	fld.Num = len(ent.Fields)
	if err := procFieldTag(f, ent, fld); err != nil {
		return err
	}

	if st, ok := sm[e.Name]; ok {
		for _, f := range st.structType.Fields.List {
			tp := tagParser{Text: f.Tag.Value}
			if err := tp.parse(); err != nil {
				return err
			}
			if _, exists := tp.Result["json"]; exists {
				fld.IsJson = true
			}
		}
	}

	ent.Fields = append(ent.Fields, fld)

	return nil
}

func (sm structsMap) procTypeEmbed(e *ast.Ident, ent *entity, imports map[string]string) error {
	if _, ok := sm[e.Name]; !ok {
		return fmt.Errorf("embedded structure %s wasn't found in package", e.Name)
	}

	subt, err := sm.procStruct(e.Name, imports)
	if err != nil {
		return err
	}

	subt.Fields[0].OpeningEmbed = append(subt.Fields[0].OpeningEmbed, e.Name)
	subt.Fields[len(subt.Fields)-1].ClosingEmbed = append(subt.Fields[0].ClosingEmbed, e.Name) //nolint:gocritic,appendAssign // it is meaningful
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
	return nil
}

func (sm structsMap) procTypeSelectorExpr(f *ast.Field, e *ast.SelectorExpr, ent *entity, _ map[string]string) error {
	fld := newField()
	fld.GoName = f.Names[0].Name
	fld.SQLName = camelCaseToSnakeCase(f.Names[0].Name)
	fld.GoType = e.Sel.Name
	fld.Num = len(ent.Fields)
	if expX, ok := e.X.(*ast.Ident); ok {
		fld.GoType = expX.Name + "." + fld.GoType
	}
	if err := procFieldTag(f, ent, fld); err != nil {
		return err
	}
	ent.Fields = append(ent.Fields, fld)
	return nil
}

func (sm structsMap) procStruct(name string, imports map[string]string) (e *entity, err error) {
	e = newEntity()
	e.Imports = imports

	if _, ok := sm[name]; !ok {
		return nil, fmt.Errorf("struct type %s not found in package", name)
	}

	e.GoName = sm[name].typeSpec.Name.Name
	e.SQLName = e.GoName
	if !*singularTablesNames {
		e.SQLName = mkPlural(e.SQLName)
	}
	e.SQLName = camelCaseToSnakeCase(e.SQLName)

	for _, f := range sm[name].structType.Fields.List {
		if err = sm.procType(f, f.Type, e, imports); err != nil {
			return
		}
	}

	if e.AutoIncrementField == nil {
		e.FieldsExcludeAutoIncrement = e.Fields
	} else {
		e.FieldsExcludeAutoIncrement = slices.DeleteFunc(slices.Clone(e.Fields), func(f *field) bool { return e.AutoIncrementField.GoName == f.GoName })
	}

	sm.procPrimaryKey(e)
	sm.procUniqIndexes(e)
	sm.procNonUniqIndexes(e)

	e.JsonFields = make([]*field, 0, len(e.FieldsExcludeAutoIncrement))
	for _, f := range e.FieldsExcludeAutoIncrement {
		if f.IsJson {
			e.JsonFields = append(e.JsonFields, f)
		}
	}

	return
}

func (sm structsMap) procPrimaryKey(e *entity) {
	for name, fields := range e.UniqIndexes {
		if name == "primary" || (e.PrimaryKey != "" && len(e.UniqIndexes[e.PrimaryKey]) > len(fields)) || e.PrimaryKey == "" {
			e.PrimaryKey = name
		}
	}
	if e.PrimaryKey != "" {
		e.FieldsExcludePrimaryKey = make([]*field, 0, len(e.Fields)-len(e.UniqIndexes[e.PrimaryKey]))
		for _, f := range e.UniqIndexes[e.PrimaryKey] {
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
}

func (sm structsMap) procUniqIndexes(e *entity) {
	shortestUniqKeyLength := len(e.Fields)
	shortestUniqKeyWOAutoIncrementLength := len(e.Fields)
	for name, fields := range e.UniqIndexes {
		var hasAutoIncrement bool
		for _, f := range fields {
			f.InIndexes = append(f.InIndexes, name)
			if e.AutoIncrementField != nil && e.AutoIncrementField.GoName != f.GoName {
				hasAutoIncrement = true
			}
		}
		if len(fields) < shortestUniqKeyLength {
			shortestUniqKeyLength = len(fields)
			e.ShortestUniqKey = name
		}
		if !hasAutoIncrement && len(fields) < shortestUniqKeyWOAutoIncrementLength {
			shortestUniqKeyWOAutoIncrementLength = len(fields)
			e.ShortestUniqWOAutoIncrementKey = name
		}
	}
}

func (sm structsMap) procNonUniqIndexes(e *entity) {
	for name, fields := range e.NonUniqIndexes {
		for _, f := range fields {
			f.InIndexes = append(f.InIndexes, name)
		}
	}
}
