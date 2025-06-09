package sqlextractor

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/kydance/sql-extractor/internal/extract"
	"github.com/kydance/sql-extractor/internal/models"
)

// Extractor is a struct that holds the raw SQL, templatized SQL, operation type,
// parameters and table information. It is used to extract information from a
// SQL string.
type Extractor struct {
	rawSQL       string                // raw SQL which needs to be extracted
	templatedSQL []string              // templatized SQL
	opType       []models.SQLOpType    // operation type: SELECT, INSERT, UPDATE, DELETE
	params       [][]any               // parameters: where conditions, order by, limit, offset
	tableInfos   [][]*models.TableInfo // table infos: Schema, Tablename
	hash         []string              // hash of the templatized SQL
}

// NewExtractor creates a new Extractor. It requires a raw SQL string.
func NewExtractor(sql string) *Extractor {
	return &Extractor{
		rawSQL:       sql,
		templatedSQL: []string{},
		opType:       []models.SQLOpType{},
		params:       [][]any{},
		tableInfos:   [][]*models.TableInfo{},
		hash:         []string{},
	}
}

// RawSQL returns the raw SQL.
func (e *Extractor) RawSQL() string { return e.rawSQL }

// SetRawSQL sets the raw SQL.
func (e *Extractor) SetRawSQL(sql string) { e.rawSQL = sql }

// TemplatizedSQL returns the templatized SQL.
func (e *Extractor) TemplatizedSQL() []string { return e.templatedSQL }

// Params returns the parameters.
func (e *Extractor) Params() [][]any { return e.params }

// TableInfos returns the table infos.
func (e *Extractor) TableInfos() [][]*models.TableInfo { return e.tableInfos }

// OpType returns the operation type.
func (e *Extractor) OpType() []models.SQLOpType { return e.opType }

// doHash calculates the hash of the templatized SQL.
func (e *Extractor) doHash(fn ...func([]byte) string) {
	e.hash = make([]string, len(e.templatedSQL))

	if len(fn) == 0 {
		fn = []func([]byte) string{func(s []byte) string {
			hash := sha256.Sum256(s)
			return hex.EncodeToString(hash[:])
		}}
	}

	for i := range e.templatedSQL {
		e.hash[i] = fn[0]([]byte(e.templatedSQL[i]))
	}
}

// TemplatizedSQLHash returns the hash of the templatized SQL.
//
// Default hash function is sha256.
func (e *Extractor) TemplatizedSQLHash(fn ...func([]byte) string) []string {
	e.doHash(fn...)
	return e.hash
}

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
	if e.templatedSQL, e.tableInfos, e.params, e.opType, err = extract.NewExtractor().Extract(e.rawSQL); err != nil {
		return err
	}
	e.doHash()

	return nil
}
