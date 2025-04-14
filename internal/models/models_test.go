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
