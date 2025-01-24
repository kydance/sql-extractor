package sqlextractor

import (
	"sql-extractor/internal/extract"
	"sql-extractor/internal/models"
)

// Extractor is a struct that holds the raw SQL, templatized SQL, operation type,
// parameters and table information. It is used to extract information from a
// SQL string.
type Extractor struct {
	rawSQL       string              // raw SQL which needs to be extracted
	templatedSQL string              // templatized SQL
	opType       models.SQLOpType    // operation type: SELECT, INSERT, UPDATE, DELETE
	params       []any               // parameters: where conditions, order by, limit, offset
	tableInfos   []*models.TableInfo // table infos: Schema, Tablename
}

// NewExtractor creates a new Extractor. It requires a raw SQL string.
func NewExtractor(sql string) *Extractor {
	return &Extractor{
		rawSQL:       sql,
		templatedSQL: "",
		params:       nil,
		tableInfos:   nil,
	}
}

// RawSQL returns the raw SQL.
func (e *Extractor) RawSQL() string { return e.rawSQL }

// SetRawSQL sets the raw SQL.
func (e *Extractor) SetRawSQL(sql string) { e.rawSQL = sql }

// TemplatizedSQL returns the templatized SQL.
func (e *Extractor) TemplatizedSQL() string { return e.templatedSQL }

// Params returns the parameters.
func (e *Extractor) Params() []any { return e.params }

// TableInfos returns the table infos.
func (e *Extractor) TableInfos() []*models.TableInfo { return e.tableInfos }

// Extract extracts information from the raw SQL string. It extracts the templatized
// SQL, parameters, table information, and operation type.
//
// Example:
//
//	extractor := NewExtractor("SELECT * FROM users WHERE id = 1")
//	err := extractor.Extract()
//	if err != nil {
//	  // handle error
//	}
//	fmt.Println(extractor.TemplatizeSQL())
func (e *Extractor) Extract() (err error) {
	e.templatedSQL, e.tableInfos, e.params, e.opType, err = extract.NewExtractor().Extract(e.rawSQL)
	if err != nil {
		return err
	}

	return nil
}
