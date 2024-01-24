package main

type entity struct {
	GoName                     string
	SQLName                    string
	Fields                     []field
	FieldsExcludePrimaryKey    []field
	FieldsExcludeAutoIncrement []field
	PrimaryIndex               string
	UniqIndexes                map[string][]*field
	NonUniqIndexes             map[string][]*field
	AutoIncrementField         *field
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

func main() {
	packageName, entities := parse()

	generate(packageName, entities)
}
