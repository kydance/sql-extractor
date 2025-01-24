package models

type TableInfo struct {
	schema    string // database
	tableName string // table name
}

// NewTableInfoWithSchemaAndName creates a new TableInfo object with a schema and a table name.
func NewTableInfoWithSchemaAndName(schema, tableName string) *TableInfo {
	return &TableInfo{
		schema:    schema,
		tableName: tableName,
	}
}

// NewTableInfo creates a new TableInfo object.
// args should be 0 or 2, and the first half are schemas, the second half are table names.
func NewTableInfo(args ...string) *TableInfo {
	if len(args) == 0 {
		return &TableInfo{}
	}

	if len(args) == 2 {
		return NewTableInfoWithSchemaAndName(args[0], args[1])
	}

	panic("invalid args: len(args) should be 0 or 2, the first half are schemas, the second half are table names.")
}

func (t *TableInfo) SetTableName(tableName string) { t.tableName = tableName }
func (t *TableInfo) TableName() string             { return t.tableName }
func (t *TableInfo) SetSchema(schema string)       { t.schema = schema }
func (t *TableInfo) Schema() string                { return t.schema }
