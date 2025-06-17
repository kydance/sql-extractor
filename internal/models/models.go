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
	SQLOperationShow    SQLOpType = "SHOW"
)

type TableInfo struct {
	templatizedSchema    string // templated schema, e.g. db_???
	templatizedTableName string // templated table name, e.g. tb_???

	schema    string // original schema, e.g. db_23
	tableName string // original table name, e.g. tb_10
}

// NewTableInfo creates a new TableInfo object.
// args should be 0 or 2 or 4
//
//   - 0: no arguments, returns an empty TableInfo.
//
//   - 2: the first is schema, the second is table name.
//
//   - 4: the first is schema, the second is table name,
//     the third is templatized schema, and the fourth is templatized table name.
func NewTableInfo(args ...string) *TableInfo {
	if len(args) == 0 {
		return &TableInfo{}
	}

	if len(args) == 2 {
		return &TableInfo{
			schema:    args[0],
			tableName: args[1],
		}
	}

	if len(args) == 4 {
		return &TableInfo{
			schema:               args[0],
			tableName:            args[1],
			templatizedSchema:    args[2],
			templatizedTableName: args[3],
		}
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

func (t *TableInfo) TemplatizedTableNameWithSchema() (string, bool) {
	if t.templatizedSchema != "" {
		return t.templatizedSchema + "." + t.templatizedTableName, true
	}
	return t.templatizedTableName, false
}
func (t *TableInfo) SetTemplatizedTableName(tableName string) { t.templatizedTableName = tableName }
func (t *TableInfo) TemplatizedTableName() string             { return t.templatizedTableName }
func (t *TableInfo) SetTemplatizedSchema(schema string)       { t.templatizedSchema = schema }
func (t *TableInfo) TemplatizedSchema() string                { return t.templatizedSchema }
