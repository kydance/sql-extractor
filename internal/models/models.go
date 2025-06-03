package models

// SQLOpType represents the type of SQL operation
type SQLOpType string

// String returns the string representation of the SQLOpType.
func (s SQLOpType) String() string { return string(s) }

const (
	SQLOperationUnknown SQLOpType = "UNKNOWN"
	SQLOperationSelect  SQLOpType = "SELECT"
	SQLOperationInsert  SQLOpType = "INSERT"
	SQLOperationUpdate  SQLOpType = "UPDATE"
	SQLOperationDelete  SQLOpType = "DELETE"
	SQLOperationExplain SQLOpType = "EXPLAIN"
)

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

	panic(
		"invalid args: len(args) should be 0 or 2, the first half are schemas, the second half are table names.",
	)
}

// TableNameWithSchema returns the table name with schema.
// If the schema is empty, it returns the table name without schema.
//
// Returns:
//   - string: the table name with schema, or the table name if the schema is empty
//   - bool: whether the schema is empty
func (t *TableInfo) TableNameWithSchema() (string, bool) {
	if t.schema != "" {
		return t.schema + "." + t.tableName, true
	}
	return t.tableName, false
}

func (t *TableInfo) SetTableName(tableName string) { t.tableName = tableName }
func (t *TableInfo) TableName() string             { return t.tableName }
func (t *TableInfo) SetSchema(schema string)       { t.schema = schema }
func (t *TableInfo) Schema() string                { return t.schema }
