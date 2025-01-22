package sqlextractor

import (
	"sql-extractor/internal/templatize"
)

type Extractor struct {
	rawSQL       string // raw SQL which needs to be templatized, extracted and etc.
	templatedSQL string // templatized SQL
	params       []any  // parameters
}

// NewExtracter creates a new Extracter. It requires a raw SQL string.
func NewExtracter(sql string) *Extractor {
	return &Extractor{
		rawSQL:       sql,
		templatedSQL: "",
		params:       nil,
	}
}

// RawSQL returns the raw SQL.
func (e *Extractor) RawSQL() string { return e.rawSQL }

// SetRawSQL sets the raw SQL.
func (e *Extractor) SetRawSQL(sql string) { e.rawSQL = sql }

// TemplatizeSQL returns the templatized SQL.
func (e *Extractor) TemplatizeSQL() string { return e.templatedSQL }

// Params returns the parameters.
func (e *Extractor) Params() []any { return e.params }

// Extract templatizes the raw SQL. It supports multiple SQL statements separated by semicolons.
// It returns an error if the SQL is invalid.
//
// Example:
//
//	extracter := NewExtracter("SELECT * FROM users WHERE id = 1")
//	err := extracter.Extract()
//	if err != nil {
//	  // handle error
//	}
//	fmt.Println(extracter.TemplatizeSQL())
func (e *Extractor) Extract() (err error) {
	e.templatedSQL, e.params, err = templatize.NewSQLTemplatizer().TemplatizeSQL(e.rawSQL)
	if err != nil {
		return err
	}

	return nil
}
