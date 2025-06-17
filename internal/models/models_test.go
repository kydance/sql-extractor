package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SQLOpType_String(t *testing.T) {
	a := assert.New(t)

	var temp SQLOpType = "SELECT"
	a.Equal("SELECT", temp.String())

	temp = SQLOperationUnknown
	a.Equal("UNKNOWN", temp.String())

	temp = SQLOperationSelect
	a.Equal("SELECT", temp.String())

	temp = SQLOperationDelete
	a.Equal("DELETE", temp.String())

	temp = SQLOperationExplain
	a.Equal("EXPLAIN", temp.String())

	temp = SQLOperationInsert
	a.Equal("INSERT", temp.String())

	temp = SQLOperationUpdate
	a.Equal("UPDATE", temp.String())
}

func TestNewTableInfo(t *testing.T) {
	a := assert.New(t)

	// Test with no arguments
	ti := NewTableInfo()
	a.NotNil(ti)
	a.Empty(ti.Schema())
	a.Empty(ti.TableName())

	// Test with 2 arguments
	ti1 := NewTableInfo("public", "users")
	a.NotNil(ti1)
	a.Equal("public", ti1.Schema())
	a.Equal("users", ti1.TableName())
	// Test with templatized table name and schema
	ti1.SetTemplatizedTableName("{{users}}")
	ti1.SetTemplatizedSchema("{{public}}")
	a.Equal("{{users}}", ti1.TemplatizedTableName())
	a.Equal("{{public}}", ti1.TemplatizedSchema())

	// Test with 4 arguments
	ti2 := NewTableInfo("public", "users", "{{public}}", "{{users}}")
	a.NotNil(ti2)
	a.Equal("public", ti2.Schema())
	a.Equal("users", ti2.TableName())
	a.Equal("{{public}}", ti2.TemplatizedSchema())
	a.Equal("{{users}}", ti2.TemplatizedTableName())

	// Test with invalid arguments
	assert.Panics(t, func() {
		NewTableInfo("invalid")
	})
}

func TestTableInfo_Methods(t *testing.T) {
	a := assert.New(t)

	// Test regular table name and schema
	ti := &TableInfo{}
	ti.SetTableName("users")
	ti.SetSchema("public")
	a.Equal("users", ti.TableName())
	a.Equal("public", ti.Schema())

	// Test templatized table name and schema
	ti.SetTemplatizedTableName("{{users}}")
	ti.SetTemplatizedSchema("{{public}}")
	a.Equal("{{users}}", ti.TemplatizedTableName())
	a.Equal("{{public}}", ti.TemplatizedSchema())

	// Test TableNameWithSchema
	name, hasSchema := ti.TableNameWithSchema()
	if hasSchema {
		a.Equal("public.users", name)
	} else {
		a.Equal("users", name)
	}

	// Test TemplatizedTableNameWithSchema
	tName, tHasSchema := ti.TemplatizedTableNameWithSchema()
	if tHasSchema {
		a.Equal("{{public}}.{{users}}", tName)
	} else {
		a.Equal("{{users}}", tName)
	}

	// Test with empty schema
	ti2 := &TableInfo{}
	ti2.SetTableName("products")
	ti2.SetTemplatizedTableName("{{products}}")

	name, hasSchema = ti2.TableNameWithSchema()
	a.False(hasSchema)
	a.Equal("products", name)

	tName, tHasSchema = ti2.TemplatizedTableNameWithSchema()
	a.False(tHasSchema)
	a.Equal("{{products}}", tName)
}
