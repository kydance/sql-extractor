package sqlextractor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtracter_RawSQL(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extracter := NewExtracter("")

	sql := "SELECT * FROM users WHERE name = 'kyden'"
	extracter.SetRawSQL(sql)
	err := extracter.Extract()
	as.Nil(err)
	as.Equal(sql, extracter.RawSQL())
}

func TestExtracter_TemplatizeSQL(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extracter := NewExtracter("")

	// single string param
	sql := "SELECT * FROM users WHERE name = 'kyden'"
	extracter.SetRawSQL(sql)
	err := extracter.Extract()
	as.Nil(err)
	as.Equal("SELECT * FROM users WHERE name eq ?", extracter.TemplatizeSQL())
}

func TestExtracter_Params(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// single string param
	sql := "SELECT * FROM users WHERE name = 'kyden'"
	extracter := NewExtracter(sql)
	err := extracter.Extract()
	as.Nil(err)
	as.Equal("SELECT * FROM users WHERE name eq ?", extracter.TemplatizeSQL())
	as.Equal([]any{"kyden"}, extracter.Params())

	// multiple params
	sql = "SELECT * FROM users WHERE name = 'kyden' AND age = 25 AND active = true"
	extracter.SetRawSQL(sql)
	err = extracter.Extract()
	as.Nil(err)
	as.Equal("SELECT * FROM users WHERE name eq ? and age eq ? and active eq ?", extracter.TemplatizeSQL())
	as.Equal([]any{"kyden", int64(25), int64(1)}, extracter.Params())

	// no params
	sql = "SELECT * FROM users"
	extracter.SetRawSQL(sql)
	err = extracter.Extract()
	as.Nil(err)
	as.Equal("SELECT * FROM users", extracter.TemplatizeSQL())
	as.Equal(0, len(extracter.Params()))
}

func TestExtracter_Extract_Error(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extracter := NewExtracter("")

	// invalid SQL syntax
	sql := "SELECT * FROM WHERE name = 'kyden'"
	extracter.SetRawSQL(sql)
	err := extracter.Extract()
	as.Error(err)

	// empty SQL
	sql = ""
	extracter.SetRawSQL(sql)
	err = extracter.Extract()
	as.Error(err)
	as.Equal("empty SQL statement", err.Error())
}

func TestExtracter_ComplexQueries(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extracter := NewExtracter("")

	// Join with conditions
	sql := "SELECT u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id WHERE u.age > 18 AND o.amount > 100.50"
	extracter.SetRawSQL(sql)
	err := extracter.Extract()
	as.Nil(err)
	as.Equal(
		"SELECT u.name, o.order_id FROM users AS u CROSS JOIN orders AS o ON u.id eq o.user_id WHERE u.age gt ? and o.amount gt ?",
		extracter.TemplatizeSQL(),
	)
	as.Equal(2, len(extracter.Params()))

	// group by and having
	sql = "SELECT department, COUNT(*) as count FROM employees WHERE salary >= 50000 GROUP BY department HAVING count > 5"
	extracter.SetRawSQL(sql)
	err = extracter.Extract()
	as.Nil(err)
	as.Equal(
		"SELECT department, COUNT(1) AS count FROM employees WHERE salary ge ? GROUP BY department HAVING count gt ?",
		extracter.TemplatizeSQL(),
	)
	as.Equal([]any{int64(50000), int64(5)}, extracter.Params())
}
