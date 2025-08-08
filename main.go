package main

import (
	"flag"
	"os/exec"
)

type entity struct {
	GoName                         string
	SQLName                        string
	Fields                         []*field
	FieldsExcludePrimaryKey        []*field
	FieldsExcludeAutoIncrement     []*field
	JsonFields                     []*field
	PrimaryKey                     string
	UniqIndexes                    map[string][]*field
	NonUniqIndexes                 map[string][]*field
	AutoIncrementField             *field
	ShortestUniqKey                string
	ShortestUniqWOAutoIncrementKey string
}

func newEntity() *entity {
	return &entity{
		UniqIndexes:    make(map[string][]*field),
		NonUniqIndexes: make(map[string][]*field),
	}
}

type field struct {
	GoName       string
	SQLName      string
	GoType       string
	IsRef        bool
	IsArray      bool
	IsJson       bool
	OpeningEmbed []string
	ClosingEmbed []string
	EmbedLevel   int
	Num          int
	InPrimaryKey bool
	InIndexes    []string
}

func newField() *field {
	return &field{}
}

var singularTablesNames = flag.Bool("singular", false, "tables names is in singular form")

func main() {
	flag.Parse()

	packageName, entities := parse()

	filename := generate(packageName, entities)

	if err := exec.Command("go", "fmt", filename).Run(); err != nil {
		panic(err)
	}
}
