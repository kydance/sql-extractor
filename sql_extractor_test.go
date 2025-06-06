package sqlextractor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kydance/sql-extractor/internal/models"
)

func TestExtractor_RawSQL(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extractor := NewExtractor("")

	sql := "SELECT * FROM users WHERE name = 'kyden'"
	extractor.SetRawSQL(sql)
	err := extractor.Extract()
	as.Nil(err)
	as.Equal(sql, extractor.RawSQL())
}

func TestExtractor_TemplatizeSQL(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extractor := NewExtractor("")

	// single string param
	sql := "SELECT * FROM users WHERE name = 'kyden'"
	extractor.SetRawSQL(sql)
	err := extractor.Extract()

	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal([]string{"SELECT * FROM users WHERE name eq ?"}, extractor.TemplatizedSQL())
	as.Equal([][]any{{"kyden"}}, extractor.Params())
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users")}}, extractor.TableInfos())
}

func TestExtractor_Params(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// single string param
	sql := "SELECT * FROM users WHERE name = 'kyden'"
	extractor := NewExtractor(sql)
	err := extractor.Extract()
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal([]string{"SELECT * FROM users WHERE name eq ?"}, extractor.TemplatizedSQL())
	as.Equal([][]any{{"kyden"}}, extractor.Params())
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users")}}, extractor.TableInfos())

	// multiple params
	sql = "SELECT * FROM users WHERE name = 'kyden' AND age = 25 AND active = true"
	extractor.SetRawSQL(sql)
	err = extractor.Extract()
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal([]string{"SELECT * FROM users WHERE name eq ? and age eq ? and active eq ?"}, extractor.TemplatizedSQL())
	as.Equal([][]any{{"kyden", int64(25), int64(1)}}, extractor.Params())
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users")}}, extractor.TableInfos())

	// no params
	sql = "SELECT * FROM users"
	extractor.SetRawSQL(sql)
	err = extractor.Extract()
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal([]string{"SELECT * FROM users"}, extractor.TemplatizedSQL())
	as.Equal(0, len(extractor.Params()[0]))
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users")}}, extractor.TableInfos())
}

func TestExtractor_TableInfos(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// single table
	sql := "SELECT * FROM users WHERE name = 'kyden'"
	extractor := NewExtractor(sql)
	err := extractor.Extract()
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal([]string{"SELECT * FROM users WHERE name eq ?"}, extractor.TemplatizedSQL())
	as.Equal([][]any{{"kyden"}}, extractor.Params())
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users")}}, extractor.TableInfos())

	// multiple tables
	sql = "SELECT * FROM users u JOIN orders o ON u.id = o.user_id WHERE u.name = 'kyden'"
	extractor.SetRawSQL(sql)
	err = extractor.Extract()
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal([]string{"SELECT * FROM users AS u CROSS JOIN orders AS o ON u.id eq o.user_id WHERE u.name eq ?"}, extractor.TemplatizedSQL())
	as.Equal([][]any{{"kyden"}}, extractor.Params())
	as.Equal([][]*models.TableInfo{
		{
			models.NewTableInfo("", "users"),
			models.NewTableInfo("", "orders"),
		},
	}, extractor.TableInfos())
}

func TestExtractor_Extract_Error(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extractor := NewExtractor("")

	// invalid SQL syntax
	sql := "SELECT * FROM WHERE name = 'kyden'"
	extractor.SetRawSQL(sql)
	err := extractor.Extract()
	as.Error(err)

	// empty SQL
	sql = ""
	extractor.SetRawSQL(sql)
	err = extractor.Extract()
	as.Error(err)
	as.Equal("empty SQL statement", err.Error())
}

func TestExtractor_ComplexQueries(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extractor := NewExtractor("")

	// Join with conditions
	sql := "SELECT u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id WHERE u.age > 18 AND o.amount > 100.50"
	extractor.SetRawSQL(sql)
	err := extractor.Extract()
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal(
		[]string{"SELECT u.name, o.order_id FROM users AS u CROSS JOIN orders AS o ON u.id eq o.user_id WHERE u.age gt ? and o.amount gt ?"},
		extractor.TemplatizedSQL(),
	)
	as.Equal(2, len(extractor.Params()[0]))
	as.Equal([][]*models.TableInfo{
		{
			models.NewTableInfo("", "users"),
			models.NewTableInfo("", "orders"),
		},
	}, extractor.TableInfos())

	t.Logf("raw SQL: %s\n Templatized SQL: %s \n TableInfos: %v \n Params: %v",
		extractor.RawSQL(), extractor.TemplatizedSQL(), extractor.TableInfos(), extractor.Params())

	// group by and having
	sql = "SELECT department, COUNT(*) as count FROM employees WHERE salary >= 50000 GROUP BY department HAVING count > 5"
	extractor.SetRawSQL(sql)
	err = extractor.Extract()
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, extractor.OpType())
	as.Equal(
		[]string{"SELECT department, COUNT(1) AS count FROM employees WHERE salary ge ? GROUP BY department HAVING count gt ?"},
		extractor.TemplatizedSQL(),
	)
	as.Equal([][]any{{int64(50000), int64(5)}}, extractor.Params())
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "employees")}}, extractor.TableInfos())
}
