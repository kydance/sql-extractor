package extract

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kydance/sql-extractor/internal/models"
)

func TestTemplatizeSQL_empty(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	template, tableInfos, params, op, err := parser.Extract("")
	as.Equal("empty SQL statement", err.Error())
	as.Equal([]string(nil), template)
	as.Equal(0, len(params))
	as.Equal(0, len(tableInfos))
	as.Equal([]models.SQLOpType(nil), op)
	as.Equal([][]*models.TableInfo(nil), tableInfos)
}

func TestTemplatizeSQL_Wildcard(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// *
	sql := "SELECT * FROM users"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{"SELECT * FROM users"}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users", "", "users")}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// u.*
	sql = "SELECT u.* FROM users u WHERE name = 'kyden'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{"SELECT u.* FROM users AS u WHERE name eq ?"}, template)
	as.Equal(1, len(params))
	as.Equal(1, len(tableInfos))
	as.Equal([]*models.TableInfo{models.NewTableInfo("", "users", "", "users")}, tableInfos[0])
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// schema
	sql = `SELECT sales.orders.* FROM sales.orders WHERE customer_id IN ( SELECT id FROM customers WHERE name LIKE 'A%' );`
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{"SELECT sales.orders.* FROM sales.orders WHERE customer_id IN ((SELECT id FROM customers WHERE name LIKE ?))"}, template)
	as.Equal(1, len(params))
	as.Equal(2, len(tableInfos[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("sales", "orders", "sales", "orders"),
		models.NewTableInfo("", "customers", "", "customers"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_eq_gt_ge_lt_le(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// =, >, >=, <, <=
	sql := "SELECT * FROM users WHERE name = 'kyden' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and uuid != 'kytedance' and create_time <> '2024-05-06 07:08:09'"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and uuid ne ? and create_time ne ?"},
		template,
	)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// !=, <>
	sql = "SELECT * FROM users WHERE name != 'Alice' AND age <> 18 AND high != 173 AND weight <> 150 and level <> 100"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name ne ? and age ne ? and high ne ? and weight ne ? and level ne ?"},
		template,
	)
	as.Equal(5, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// and >=
	sql = "select * from tbGMallCfmH5UserDayLottery where  sOpenid = 'owXVa5LsfyqACPIbQpEFPYLRvUNo' and dtCommitTime >=  '2024-11-26 00:00:00' and iStatus = 1"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM tbGMallCfmH5UserDayLottery WHERE sOpenid eq ? and dtCommitTime ge ? and iStatus eq ?"},
		template,
	)
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "tbGMallCfmH5UserDayLottery", "", "tbGMallCfmH5UserDayLottery"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_between_and(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// between and date
	sql := "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02'"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ?"},
		template,
	)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// between and date
	sql = "select * from users WHERE create_time between '2024-05-06 07:08:09' and '2024-05-07 07:08:09'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE create_time BETWEEN ? AND ?"},
		template,
	)
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_in(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// IN (v1, v2, ...)
	sql := "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?)"},
		template,
	)
	as.Equal(10, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// NOT IN (...)
	sql = `SELECT * FROM users WHERE name = 'kyden' AND uuid not in ('kytedance', 'kydance')`
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and uuid NOT IN (?, ?)"},
		template,
	)
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// IN (SELECT ...)
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (SELECT id FROM users WHERE create_time between '2021-01-01' and '2021-01-02')"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN ((SELECT id FROM users WHERE create_time BETWEEN ? AND ?))"},
		template,
	)
	as.Equal(9, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// NOT IN (SELECT ...)
	sql = "SELECT * FROM users WHERE name = 'kyden' and uuid NOT in (SELECT uuid FROM users WHERE create_time between '2021-01-01' and '2021-01-02')"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and uuid NOT IN ((SELECT uuid FROM users WHERE create_time BETWEEN ? AND ?))"},
		template,
	)
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_like(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// like 'Kyden%'
	sql := "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%'"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ?"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// like '%Kyden'
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like '%Kyden'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ?"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// like '%Kyden%'
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like '%Kyden%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ?"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// like '_yden%'
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like '_yden%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ?"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// like 'Kyden_'
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden_'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ?"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Not Like 'Kyden%'
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name not like 'Kyden%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name NOT LIKE ?"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// like 'Kyden%' or like '%Kyden' or like '%Kyden%' or not like '_yden' or not like 'Kyden_'
	sql = "SELECT * FROM users WHERE name like 'Kyden%' or name like '%Kyden' or name like '%Kyden%' or name not like '_yden' or name not like 'Kyden_'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name LIKE ? or name LIKE ? or name LIKE ? or name NOT LIKE ? or name NOT LIKE ?"},
		template,
	)
	as.Equal(5, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_GroupBy(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Group By name
	sql := "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Group By name, age
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	sql = "SELECT * FROM users WHERE name like 'kyden%' AND age > 18 AND high >= 173 AND weight < 150 GROUP BY name, age"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name LIKE ? and age gt ? and high ge ? and weight lt ? GROUP BY name, age"},
		template,
	)
	as.Equal(4, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_OrderBy(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Order By name
	sql := "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' ORDER BY name "
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? ORDER BY name"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	//
	sql = "SELECT * FROM users WHERE age > 18 AND high >= 173 AND weight < 150 and create_time between '2021-01-01' and '2021-01-02' and name like 'Kyden%' ORDER BY name "
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE age gt ? and high ge ? and weight lt ? and create_time BETWEEN ? AND ? and name LIKE ? ORDER BY name"},
		template,
	)
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Order By name, age
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' ORDER BY name, age"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? ORDER BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Order By name DESC
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' ORDER BY name DESC"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? ORDER BY name DESC"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Order By name, age DESC
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' ORDER BY name, age DESC"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? ORDER BY name, age DESC"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Order By name DESC, age
	sql = "SELECT * FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' ORDER BY name DESC, age"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? ORDER BY name DESC, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_AggregateFunc_AS(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// Count
	sql := "SELECT count(*) FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err := NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Count(Distinct ...)
	sql = "SELECT count(distinct age) FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(DISTINCT age) FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Count(distinct ...) as ...
	sql = "SELECT count(distinct age) as cnt FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(DISTINCT age) AS cnt FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Sum
	sql = "SELECT sum(age) FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT sum(age) FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Avg
	sql = "SELECT avg(age) FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT avg(age) FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Count, Sum, Max
	sql = "SELECT count(*), sum(age), max(age) FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1), sum(age), max(age) FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// AS
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	//
	sql = "SELECT count(distinct age) as cnt FROM users WHERE high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2022-01-01'"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT count(DISTINCT age) AS cnt FROM users WHERE high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ?"},
		template,
	)
	as.Equal(5, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_Limit(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// Limit 10
	sql := "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age LIMIT 10"
	template, tableInfos, params, op, err := NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age LIMIT ?"},
		template,
	)
	as.Equal(12, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Limit 10, 20
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age LIMIT 10, 20"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age LIMIT ?, ?"},
		template,
	)
	as.Equal(13, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// LIMIT 10 OFFSET 20
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age LIMIT 10 OFFSET 20"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age LIMIT ?, ?"},
		template,
	)
	as.Equal(13, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_Having(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Having sum(age) > 100
	sql := "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 LIMIT 10, 20"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(14, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Having sum(age) > 100 AND max(age) < 100
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 AND max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? and max(age) lt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Having age > 18 and sum(age) > 100 OR max(age) < 100
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING age > 18 and sum(age) > 100 OR max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING age gt ? and sum(age) gt ? or max(age) lt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(16, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	//
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2024-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING age > 18 and sum(age) > 100 OR max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING age gt ? and sum(age) gt ? or max(age) lt ? LIMIT ?, ?"},
		template)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Having aggregate functions
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2024-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING age > 18 and sum(age) > 100 OR max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING age gt ? and sum(age) gt ? or max(age) lt ? LIMIT ?, ?"},
		template)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Having aggregate functions
	sql = "SELECT count(*) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2024-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT count(1) AS cnt, sum(age) AS sum_age, max(age) AS max_age FROM users WHERE age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? LIMIT ?, ?"},
		template)
	as.Equal(13, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// HAVING aggregate functions
	sql = `SELECT department,
       AVG(salary) as avg_salary,
       COUNT(*) as employee_count
FROM employees
GROUP BY department
HAVING AVG(salary) > 50000 AND COUNT(*) > 10`
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT department, AVG(salary) AS avg_salary, COUNT(1) AS employee_count FROM employees GROUP BY department HAVING AVG(salary) gt ? and COUNT(1) gt ?"},
		template)
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "employees", "", "employees"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_Join(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// subquery
	sql := "SELECT * FROM (SELECT * FROM users) AS t1 WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 OR max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM (SELECT * FROM users) AS t1 WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? or max(age) lt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 多层 JOIN
	sql = "SELECT * FROM users u LEFT JOIN roles r ON u.id = r.user_id LEFT JOIN ages a ON u.id = a.age_id WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 OR max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users AS u LEFT JOIN roles AS r ON u.id eq r.user_id LEFT JOIN ages AS a ON u.id eq a.age_id WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? or max(age) lt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "roles", "", "roles"),
		models.NewTableInfo("", "ages", "", "ages"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// t1 left join t2 join t3
	sql = `SELECT t1.*, t2.name
		         FROM schema1.table1 t1
		         LEFT JOIN (SELECT * FROM table2) t2 ON t1.id = t2.id
		         INNER JOIN table3 t3 ON t2.id = t3.id`
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT t1.*, t2.name FROM schema1.table1 AS t1 LEFT JOIN (SELECT * FROM table2) AS t2 ON t1.id eq t2.id CROSS JOIN table3 AS t3 ON t2.id eq t3.id"},
		template,
	)
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("schema1", "table1", "schema1", "table1"),
		models.NewTableInfo("", "table2", "", "table2"),
		models.NewTableInfo("", "table3", "", "table3"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 复杂 JOIN
	sql = `SELECT t1.*, t2.name
		         FROM schema1.table1 t1
		         LEFT JOIN (SELECT * FROM table2) t2 ON t1.id = t2.id
		         JOIN table3 t3 ON t2.id = t3.id`
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT t1.*, t2.name FROM schema1.table1 AS t1 LEFT JOIN (SELECT * FROM table2) AS t2 ON t1.id eq t2.id CROSS JOIN table3 AS t3 ON t2.id eq t3.id"},
		template,
	)
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("schema1", "table1", "schema1", "table1"),
		models.NewTableInfo("", "table2", "", "table2"),
		models.NewTableInfo("", "table3", "", "table3"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Join
	sql = "SELECT * FROM users u LEFT JOIN roles r ON u.id = r.user_id WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 OR max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users AS u LEFT JOIN roles AS r ON u.id eq r.user_id WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? or max(age) lt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "roles", "", "roles"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	//
	sql = `SELECT t1.*, t2.name FROM schema1.table1 t1 LEFT JOIN (SELECT * FROM table2) t2 ON t1.id = t2.id JOIN table3 t3 ON t2.id = t3.id WHERE t1.id = 1 and t2.name = 'Kyden' and t3.name = 'kytedance' and t3.create_time between '2021-01-01' and '2021-01-02' and t3.age > 18 GROUP BY t1.id HAVING sum(t1.age) > 100 OR max(t1.age) < 100 LIMIT 10, 20`

	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT t1.*, t2.name FROM schema1.table1 AS t1 LEFT JOIN (SELECT * FROM table2) AS t2 ON t1.id eq t2.id CROSS JOIN table3 AS t3 ON t2.id eq t3.id WHERE t1.id eq ? and t2.name eq ? and t3.name eq ? and t3.create_time BETWEEN ? AND ? and t3.age gt ? GROUP BY t1.id HAVING sum(t1.age) gt ? or max(t1.age) lt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(10, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("schema1", "table1", "schema1", "table1"),
		models.NewTableInfo("", "table2", "", "table2"),
		models.NewTableInfo("", "table3", "", "table3"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_SELECT_DISTINCT(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// SELECT DISTINCT
	sql := "SELECT DISTINCT name, age FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT DISTINCT name, age FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age"},
		template,
	)
	as.Equal(11, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_Insert(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// INSERT INTO table_name (column1, column2, ...) VALUES (value1, value2, ...);
	sql := "INSERT INTO users (name, age, high, weight, level, create_time) VALUES ('Alice', 18, 173, 150, 100, '2021-01-01 00:00:00')"
	template, tableInfos, params, op, err := NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age, high, weight, level, create_time) VALUES (?, ?, ?, ?, ?, ?)"},
		template,
	)
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// INSERT INTO table_name VALUES (value1, value2, ...);
	sql = "INSERT INTO users VALUES ('Alice', 18, 173, 150, 100, '2021-01-01 00:00:00')"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users VALUES (?, ?, ?, ?, ?, ?)"},
		template)
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// INSERT INTO table_name (column1, column2, ...) VALUES (value1_1, value1_2, ...), (value2_1, value2_2, ...), ...;
	sql = "INSERT INTO users (name, age, high, weight, level, create_time) VALUES ('Alice', 18, 173, 150, 100, '2021-01-01 00:00:00'), ('Bob', 20, 180, 160, 100, '2021-01-02 00:00:00')"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age, high, weight, level, create_time) VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)"},
		template,
	)
	as.Equal(12, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// INSERT INTO ... SELECT ...
	sql = "INSERT INTO users (name, age, high, weight, level, create_time) SELECT name, age, high, weight, level, create_time FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 OR max(age) < 100 LIMIT 10, 20"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age, high, weight, level, create_time) SELECT name, age, high, weight, level, create_time FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? or max(age) lt ? LIMIT ?, ?"},
		template,
	)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// INSERT IGNORE INTO table_name (column1, column2, ...) VALUES (value1, value2, ...);
	sql = "INSERT IGNORE INTO users (name, age, high, weight, level, create_time) VALUES ('Alice', 18, 173, 150, 100, '2021-01-01 00:00:00')"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT IGNORE INTO users (name, age, high, weight, level, create_time) VALUES (?, ?, ?, ?, ?, ?)"},
		template,
	)
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// INSERT INTO ... SELECT ... ON DUPLICATE KEY UPDATE ... VALUES(...)...
	sql = "INSERT INTO users (name, age, high, weight, level, create_time) SELECT name, age, high, weight, level, create_time FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 OR max(age) < 100 LIMIT 10, 20 ON DUPLICATE KEY UPDATE name = VALUES(name), age = VALUES(age), high = VALUES(high), weight = VALUES(weight), level = VALUES(level), create_time = VALUES(create_time)"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age, high, weight, level, create_time) SELECT name, age, high, weight, level, create_time FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? or max(age) lt ? LIMIT ?, ? ON DUPLICATE KEY UPDATE name eq VALUES(name), age eq VALUES(age), high eq VALUES(high), weight eq VALUES(weight), level eq VALUES(level), create_time eq VALUES(create_time)"},
		template,
	)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// INSERT INTO ... SELECT ... ON DUPLICATE KEY UPDATE ...
	sql = "INSERT INTO users (name, age, high, weight, level, create_time) SELECT name, age, high, weight, level, create_time FROM users WHERE name = 'Alice' AND age > 18 AND high >= 173 AND weight < 150 and level <= 100 and create_time between '2021-01-01' and '2021-01-02' and id in (1, 2, 3) and name like 'Kyden%' GROUP BY name, age HAVING sum(age) > 100 OR max(age) < 100 LIMIT 10, 20 ON DUPLICATE KEY UPDATE name = name, age = age, high = high, weight = weight, level = level, create_time = create_time"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age, high, weight, level, create_time) SELECT name, age, high, weight, level, create_time FROM users WHERE name eq ? and age gt ? and high ge ? and weight lt ? and level le ? and create_time BETWEEN ? AND ? and id IN (?, ?, ?) and name LIKE ? GROUP BY name, age HAVING sum(age) gt ? or max(age) lt ? LIMIT ?, ? ON DUPLICATE KEY UPDATE name eq name, age eq age, high eq high, weight eq weight, level eq level, create_time eq create_time"},
		template,
	)
	as.Equal(15, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// INSERT INTO ... () VALUES ()
	sql = "INSERT INTO users () VALUES ()"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users VALUES ()"},
		template)
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}},
		tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	//
	sql = `INSERT INTO users
    (name, age, high, weight, level, create_time)
SELECT name, age, high, weight, level, create_time FROM users WHERE name = 'kyden'
ON DUPLICATE KEY UPDATE
    name = VALUES(name), age = VALUES(age), high = VALUES(high), weight = VALUES(weight), level = VALUES(level), create_time = VALUES(create_time)`
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{`INSERT INTO users (name, age, high, weight, level, create_time) SELECT name, age, high, weight, level, create_time FROM users WHERE name eq ? ON DUPLICATE KEY UPDATE name eq VALUES(name), age eq VALUES(age), high eq VALUES(high), weight eq VALUES(weight), level eq VALUES(level), create_time eq VALUES(create_time)`},
		template,
	)
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)
}

func TestTemplatizeSQL_Update(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// UPDATE table_name SET ... WHERE ...
	sql := "UPDATE users SET name = 'Alice', age = 18, high = 173, weight = 150, level = 100, create_time = '2021-01-01 00:00:00' WHERE id = 1"
	template, tableInfos, params, op, err := NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{
			"UPDATE users SET name eq ?, age eq ?, high eq ?, weight eq ?, level eq ?, create_time eq ? WHERE id eq ?",
		},
		template,
	)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE table_name SET ...
	sql = "UPDATE users SET name = 'Alice', age = 18, high = 173, weight = 150, level = 100, create_time = '2021-01-01 00:00:00'"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{
			"UPDATE users SET name eq ?, age eq ?, high eq ?, weight eq ?, level eq ?, create_time eq ?",
		},
		template,
	)
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE table_name SET ... WHERE ... ORDER BY ... LIMIT ...
	sql = "UPDATE users SET name = 'Alice', age = 18, high = 173, weight = 150, level = 100, create_time = '2021-01-01 00:00:00' WHERE id = 1 ORDER BY name, age LIMIT 20"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users SET name eq ?, age eq ?, high eq ?, weight eq ?, level eq ?, create_time eq ? WHERE id eq ? ORDER BY name, age LIMIT ?"},
		template,
	)
	as.Equal(8, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE with subquery
	sql = "UPDATE users SET name = 'Alice', age = 18, high = 173, weight = 150, level = 100, create_time = '2021-01-01 00:00:00' WHERE id = (SELECT id FROM users WHERE name = 'Alice')"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users SET name eq ?, age eq ?, high eq ?, weight eq ?, level eq ?, create_time eq ? WHERE id eq (SELECT id FROM users WHERE name eq ?)"},
		template,
	)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE with subquery
	sql = "UPDATE users SET name = 'Alice', age = (SELECT age FROM users WHERE name = 'Alice'), high = 173, weight = 150, level = (SELECT level FROM users WHERE name = 'Alice'), create_time = '2021-01-01 00:00:00' WHERE id = (SELECT id FROM users WHERE name = 'Alice') ORDER BY name, age DESC LIMIT 20"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users SET name eq ?, age eq (SELECT age FROM users WHERE name eq ?), high eq ?, weight eq ?, level eq (SELECT level FROM users WHERE name eq ?), create_time eq ? WHERE id eq (SELECT id FROM users WHERE name eq ?) ORDER BY name, age DESC LIMIT ?"},
		template,
	)
	as.Equal(8, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE with subquery, order by, limit
	sql = "UPDATE users SET name = 'Alice', age = (SELECT age FROM users WHERE name = 'Alice'), high = 173, weight = 150, level = (SELECT level FROM users WHERE name = 'Alice'), create_time = '2021-01-01 00:00:00' WHERE id = (SELECT id FROM users WHERE name = 'Alice') ORDER BY name, age DESC LIMIT 20"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users SET name eq ?, age eq (SELECT age FROM users WHERE name eq ?), high eq ?, weight eq ?, level eq (SELECT level FROM users WHERE name eq ?), create_time eq ? WHERE id eq (SELECT id FROM users WHERE name eq ?) ORDER BY name, age DESC LIMIT ?"},
		template,
	)
	as.Equal(8, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE as
	sql = "UPDATE users as u SET name = 'Alice', age = 18, high = 173, weight = 150, level = 100, create_time = '2021-01-01 00:00:00' WHERE id = 1"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users AS u SET name eq ?, age eq ?, high eq ?, weight eq ?, level eq ?, create_time eq ? WHERE id eq ?"},
		template,
	)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE as
	sql = "UPDATE users as u SET u.name = 'Alice', u.age = 18, u.high = 173, u.weight = 150, u.level = 100, u.create_time = '2021-01-01 00:00:00' WHERE u.id = 1"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users AS u SET u.name eq ?, u.age eq ?, u.high eq ?, u.weight eq ?, u.level eq ?, u.create_time eq ? WHERE u.id eq ?"},
		template,
	)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE with JOIN
	sql = "UPDATE users as u1 JOIN users as u2 ON u1.manager_id = u2.id SET u1.name = u2.name, u1.age = u2.age, u1.high = u2.high, u1.weight = u2.weight, u1.level = u2.level, u1.create_time = u2.create_time WHERE u1.id = 1"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users AS u1 CROSS JOIN users AS u2 ON u1.manager_id eq u2.id SET u1.name eq u2.name, u1.age eq u2.age, u1.high eq u2.high, u1.weight eq u2.weight, u1.level eq u2.level, u1.create_time eq u2.create_time WHERE u1.id eq ?"},
		template,
	)
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	//
	sql = "UPDATE users as u SET name = 'kyden', age = 18, high = 175, weight = 142, level = 100, create_time = '2021-01-01 00:00:00' WHERE uuid = (SELECT uuid FROM users WHERE name = 'kyden')"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users AS u SET name eq ?, age eq ?, high eq ?, weight eq ?, level eq ?, create_time eq ? WHERE uuid eq (SELECT uuid FROM users WHERE name eq ?)"},
		template,
	)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	//
	sql = "UPDATE users as u1 JOIN users as u2 ON u1.manager_id = u2.id SET u1.name = 'kyden', u1.age = 18, u1.high = 175, u1.weight = u2.weight, u1.level = u2.level, u1.create_time = u2.create_time WHERE u1.uuid = 'kytedance'"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users AS u1 CROSS JOIN users AS u2 ON u1.manager_id eq u2.id SET u1.name eq ?, u1.age eq ?, u1.high eq ?, u1.weight eq u2.weight, u1.level eq u2.level, u1.create_time eq u2.create_time WHERE u1.uuid eq ?"},
		template,
	)
	as.Equal(4, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)
}

func TestTemplatizeSQL_ComplexUpdate(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	paser := NewExtractor()

	// UPDATE with multiple tables
	sql := "UPDATE users as u1, users as u2 SET u1.name = 'Alice', u1.age = 18, u1.high = 173, u1.weight = 150, u2.level = 100, u2.create_time = '2021-01-01 00:00:00'"
	template, tableInfos, params, op, err := paser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users AS u1 CROSS JOIN users AS u2 SET u1.name eq ?, u1.age eq ?, u1.high eq ?, u1.weight eq ?, u2.level eq ?, u2.create_time eq ?"},
		template)
	as.Equal(6, len(params[0]))
	as.Equal(
		[][]*models.TableInfo{{
			models.NewTableInfo("", "users", "", "users"),
		}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	//
	sql = "UPDATE users as u1, users as u2 SET u1.name = 'kyden', u1.age = 18, u1.high = 175, u1.weight = u2.weight, u1.level = u2.level, u1.create_time = u2.create_time WHERE u1.uuid = 'kytedance'"
	template, tableInfos, params, op, err = paser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users AS u1 CROSS JOIN users AS u2 SET u1.name eq ?, u1.age eq ?, u1.high eq ?, u1.weight eq u2.weight, u1.level eq u2.level, u1.create_time eq u2.create_time WHERE u1.uuid eq ?"},
		template,
	)
	as.Equal(4, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)
}

func TestTemplatizeSQL_case_when(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	paser := NewExtractor()

	// UPDATE with simple CASE
	sql := `UPDATE users SET name = CASE id WHEN 1 THEN 'kyden' ELSE 'kytedance' END, age = CASE id WHEN 1 THEN 18 ELSE 20 END WHERE id = 1`
	template, tableInfos, params, op, err := paser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users SET name eq CASE id WHEN ? THEN ? ELSE ? END, age eq CASE id WHEN ? THEN ? ELSE ? END WHERE id eq ?"},
		template)
	as.Equal(7, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// UPDATE with searched CASE
	sql = `UPDATE users SET name = CASE WHEN id = 1 THEN 'Alice' WHEN id = 2 THEN 'Bob' ELSE 'Unknown' END WHERE id < 10`
	template, tableInfos, params, op, err = paser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"UPDATE users SET name eq CASE WHEN id eq ? THEN ? WHEN id eq ? THEN ? ELSE ? END WHERE id lt ?"},
		template)
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)
}

func TestTemplatizeSQL_Delete(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// DELETE FROM table_name WHERE ...
	sql := "DELETE FROM users WHERE id = 1"
	template, tableInfos, params, op, err := NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE FROM users WHERE id eq ?"},
		template)
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)

	// DELETE FROM ...
	sql = "DELETE FROM users"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE FROM users"},
		template)
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)

	// DELETE t1 FROM tb1 t1 INNER JOIN tb2 t2 ON t1.id = t2.id WHERE t1.id = 1
	sql = "DELETE u FROM users u INNER JOIN roles r ON u.id = r.user_id WHERE u.id = 1"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE u FROM users AS u CROSS JOIN roles AS r ON u.id eq r.user_id WHERE u.id eq ?"},
		template,
	)
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "u", "", "u"),
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "roles", "", "roles"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)

	// DELETE FROM table_name WHERE ... LIMIT ...
	sql = "DELETE FROM users WHERE id = 1 LIMIT 10"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE FROM users WHERE id eq ? LIMIT ?"},
		template)
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)

	// DELETE FROM table_name ORDER BY ... LIMIT ...
	sql = "DELETE FROM users ORDER BY name, age LIMIT 10"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE FROM users ORDER BY name, age LIMIT ?"},
		template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)

	// DELETE FROM table_name WHERE id IN (SELECT ...)
	sql = "DELETE FROM users WHERE id IN (SELECT id FROM roles WHERE create_time > '2021-01-01 00:00:00')"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE FROM users WHERE id IN ((SELECT id FROM roles WHERE create_time gt ?))"},
		template,
	)
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "roles", "", "roles"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)

	// DELETE FROM t1, t2 FROM tb1 t1 INNER JOIN tb2 t2 ON t1.id = t2.id WHERE t1.id = 1
	sql = "DELETE u, r FROM users u INNER JOIN roles r ON u.id = r.user_id WHERE u.id = 1"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE u, r FROM users AS u CROSS JOIN roles AS r ON u.id eq r.user_id WHERE u.id eq ?"},
		template,
	)
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "u", "", "u"),
		models.NewTableInfo("", "r", "", "r"),
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "roles", "", "roles"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)

	// FIXME delete alias
	sql = "DELETE u FROM users u INNER JOIN roles r ON u.id = r.user_id WHERE u.uuid = 'kytedance'"
	template, tableInfos, params, op, err = NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"DELETE u FROM users AS u CROSS JOIN roles AS r ON u.id eq r.user_id WHERE u.uuid eq ?"},
		template,
	)
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "u", "", "u"),
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "roles", "", "roles"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)
}

func TestTemplatizeSQL_complex(t *testing.T) {
	t.Parallel()
	as := assert.New(t)

	// UPDATE with multiple tables
	sql := "INSERT INTO tb6 (`sKey`,`sBody`,`dtCreateTime`,`iAppId`,`sModule`,`iActId`,`sUid`,`sBizCode`,`iVersion`,`sAction`) VALUES ('order_LOL-2','','2024-11-26 21:23:07','1001','ConfirmTradi','2345','12345678','lzjadd','1','{\"default_ip\":\"\",\"l5_cmd\":\"1234\",\"l5_mod\":\"2345\",\"nobody\":\"1\",\"times\":\"0\",\"url\":\"http://tencent-cloud.net/red_dot?red_type=1&_t=1731655024\"}');"

	template, tableInfos, params, op, err := NewExtractor().Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO tb6 (sKey, sBody, dtCreateTime, iAppId, sModule, iActId, sUid, sBizCode, iVersion, sAction) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"},
		template)
	as.Equal(10, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "tb6", "", "tb6"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)
}

func TestTemplatizeSQL_MultipleStatements(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	psr := NewExtractor()

	// Test multiple SQL statements
	sql := `INSERT INTO users (name, age) VALUES ('Alice', 25);
		UPDATE users SET age = 26 WHERE name = 'Alice';
		DELETE FROM users WHERE name = 'Alice' AND age > 25;`
	template, tableInfos, params, op, err := psr.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{
			"INSERT INTO users (name, age) VALUES (?, ?)",
			"UPDATE users SET age eq ? WHERE name eq ?",
			"DELETE FROM users WHERE name eq ? and age gt ?",
		},
		template,
	)
	as.Equal(3, len(params))
	as.Equal("Alice", params[0][0])
	as.Equal(int64(25), params[0][1])
	as.Equal("Alice", params[1][0])
	as.Equal(int64(25), params[1][1])
	as.Equal("Alice", params[2][0])
	as.Equal(int64(25), params[2][1])
	as.Equal([][]*models.TableInfo{
		{models.NewTableInfo("", "users", "", "users")},
		{models.NewTableInfo("", "users", "", "users")},
		{models.NewTableInfo("", "users", "", "users")},
	}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert, models.SQLOperationUpdate, models.SQLOperationDelete}, op)

	// Test error case with invalid SQL in the middle
	sql = `INSERT INTO users (name, age) VALUES ('Bob', 30);
		INVALID SQL STATEMENT;
		DELETE FROM users WHERE name = 'Bob';`
	_, _, _, _, err = psr.Extract(sql)
	as.Error(err)

	// Test error case with invalid SQL at the end
	sql = "INSERT INTO tbTradiQueueRT_6 (`sKey`,`sBody`,`dtCreateTime`,`iAppId`,`sModule`,`iActId`,`sUid`,`sBizCode`,`iVersion`,`sAction`) VALUES ('order_L-2783-567_2','','2024-11-26 21:23:07','101','ConfirmTradi','224','456789012','l','1','{\"default_ip\":\"\",\"l5_cmd\":\"123\",\"l5_mod\":\"2345\",\"nobody\":\"1\",\"times\":\"0\",\"url\":\"http://teeest.tencent-cloud.net/red_dot?red_type=1&_t=1731655024\"}');INSERT INTO tbTradiQueueUK (`dtCreateTime`,`iActId`,`iVersion`,`sBody`,`sModule`,`iAppId`,`sAction`,`sBizCode`,`sKey`,`sUid`) VALUES ('2024-11-26 21:23:07','224','1','','ConfirmTradi','1','[{\"default_ip\":\"\",\"l5_cmd\":\"123\",\"l5_mod\":\"2345\",\"nobody\":\"1\",\"times\":\"0\",\"url\":\"http://teeest.tencent-cloud.net/red_dot?\\u0026red_type=1\\u0026_t=1731655024\"}]','lzjd','order_L-2024111553-527_2','42712345')"
	template, tableInfos, params, op, err = psr.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{
			"INSERT INTO tbTradiQueueRT_? (sKey, sBody, dtCreateTime, iAppId, sModule, iActId, sUid, sBizCode, iVersion, sAction) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"INSERT INTO tbTradiQueueUK (dtCreateTime, iActId, iVersion, sBody, sModule, iAppId, sAction, sBizCode, sKey, sUid) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		},
		template)
	as.Equal(2, len(params))
	as.Equal(10, len(params[0]))
	as.Equal(10, len(params[1]))
	as.Equal([][]*models.TableInfo{
		{models.NewTableInfo("", "tbTradiQueueRT_6", "", "tbTradiQueueRT_?")},
		{models.NewTableInfo("", "tbTradiQueueUK", "", "tbTradiQueueUK")},
	}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert, models.SQLOperationInsert}, op)
}

func TestTemplatizeSQL_Parentheses(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// 1. 简单括号表达式
	sql := "SELECT * FROM users WHERE (age > 18)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE (age gt ?)"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 2. 复杂括号表达式
	sql = "SELECT * FROM users WHERE (age > 18 AND (height > 170 OR weight < 65))"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE (age gt ? and (height gt ? or weight lt ?))"},
		template)
	as.Equal(1, len(params))
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 3. 带有 IN 的括号表达式
	sql = "SELECT * FROM users WHERE (id IN (1, 2, 3) OR (age > 18 AND height > 170))"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE (id IN (?, ?, ?) or (age gt ? and height gt ?))"},
		template)
	as.Equal(1, len(params))
	as.Equal(5, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 4. 带有子查询的括号表达式
	sql = "SELECT * FROM users WHERE (id IN (SELECT id FROM roles) OR (age > 18))"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE (id IN ((SELECT id FROM roles)) or (age gt ?))"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "roles", "", "roles"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 5. 带有计算的括号表达式
	sql = "SELECT *, (price * quantity) as total FROM orders WHERE (price * quantity) > 1000"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT *, (price mul quantity) AS total FROM orders WHERE (price mul quantity) gt ?"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 6. 带有 CASE WHEN 的括号表达式
	sql = "SELECT * FROM users WHERE (CASE WHEN age > 18 THEN 'adult' ELSE 'minor' END) = 'adult'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE (CASE WHEN age gt ? THEN ? ELSE ? END) eq ?"},
		template)
	as.Equal(1, len(params))
	as.Equal(4, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 7. 带有聚合函数的括号表达式
	sql = "SELECT *, (COUNT(*) + SUM(quantity)) as total FROM orders GROUP BY user_id HAVING (COUNT(*) + SUM(quantity)) > 100"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT *, (COUNT(1) plus SUM(quantity)) AS total FROM orders GROUP BY user_id HAVING (COUNT(1) plus SUM(quantity)) gt ?"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 聚合函数中
	sql = "SELECT (COUNT(*) + sum(2.0) + avg(3.0)) as total FROM orders"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT (COUNT(1) plus sum(2.0) plus avg(3.0)) AS total FROM orders"},
		template)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_FuncCall(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// 测试日期/时间函数
	sql := "SELECT DATE_FORMAT(create_time, '%Y-%m-%d') as date, COUNT(*) as count FROM users WHERE create_time > NOW()"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT DATE_FORMAT(create_time, ?) AS date, COUNT(1) AS count FROM users WHERE create_time gt NOW()"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 测试字符串函数
	sql = "SELECT * FROM users WHERE LOWER(name) = 'admin' AND SUBSTRING(email, 1, 3) = 'abc' AND CONCAT(first_name, ' ', last_name) LIKE '%John%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE LOWER(name) eq ? and SUBSTRING(email, ?, ?) eq ? and CONCAT(first_name, ?, last_name) LIKE ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 测试数学函数
	sql = "SELECT id, ROUND(price, 2) as price, ABS(score) as abs_score FROM products WHERE CEIL(rating) >= 4"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT id, ROUND(price, ?) AS price, ABS(score) AS abs_score FROM products WHERE CEIL(rating) ge ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "products", "", "products"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 测试嵌套函数调用
	sql = "SELECT * FROM orders WHERE YEAR(create_time) = YEAR(NOW()) AND MONTH(create_time) = MONTH(NOW())"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM orders WHERE YEAR(create_time) eq YEAR(NOW()) and MONTH(create_time) eq MONTH(NOW())"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 测试在GROUP BY和HAVING中使用函数
	sql = "SELECT DATE_FORMAT(create_time, '%Y-%m-%d') as date, COUNT(*) as count FROM orders GROUP BY DATE_FORMAT(create_time, '%Y-%m-%d') HAVING COUNT(*) > 100"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT DATE_FORMAT(create_time, ?) AS date, COUNT(1) AS count FROM orders GROUP BY DATE_FORMAT(create_time, ?) HAVING COUNT(1) gt ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 复杂场景
	sql = "SELECT DATE_FORMAT(create_time, '%Y-%m-%d') as date, COUNT(*) as count FROM orders WHERE YEAR(create_time) = YEAR(NOW()) AND MONTH(create_time) = MONTH(NOW()) GROUP BY DATE_FORMAT(create_time, '%Y-%m-%d') HAVING COUNT(*) > 100"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT DATE_FORMAT(create_time, ?) AS date, COUNT(1) AS count FROM orders WHERE YEAR(create_time) eq YEAR(NOW()) and MONTH(create_time) eq MONTH(NOW()) GROUP BY DATE_FORMAT(create_time, ?) HAVING COUNT(1) gt ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_UnaryOperation(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test negative number
	sql := "SELECT -age, -1 FROM users WHERE score > -100"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT minus age, minus ? FROM users WHERE score gt minus ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test NOT operation
	sql = "SELECT * FROM users WHERE NOT is_deleted AND NOT (age < 18 OR level > 100)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE not is_deleted and not (age lt ? or level gt ?)"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test bitwise NOT
	sql = "SELECT ~flags FROM users WHERE ~permission_bits = 0"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT bitneg flags FROM users WHERE bitneg permission_bits eq ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_IsNull(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test IS NULL
	sql := "SELECT * FROM users WHERE email IS NULL AND phone IS NULL"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE email IS NULL and phone IS NULL"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test IS NOT NULL
	sql = "SELECT * FROM users WHERE email IS NOT NULL AND last_login IS NOT NULL"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE email IS NOT NULL and last_login IS NOT NULL"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test mixed with other conditions
	sql = "SELECT * FROM users WHERE email IS NULL AND age > 18 AND status IS NOT NULL"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE email IS NULL and age gt ? and status IS NOT NULL"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_Exists(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test EXISTS
	sql := "SELECT * FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE EXISTS ((SELECT ? FROM orders WHERE orders.user_id eq users.id))"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test NOT EXISTS
	sql = "SELECT * FROM users WHERE NOT EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id AND total > 100)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE NOT EXISTS ((SELECT ? FROM orders WHERE orders.user_id eq users.id and total gt ?))"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test EXISTS with complex conditions
	sql = "SELECT * FROM users WHERE age > 18 AND EXISTS (SELECT 1 FROM orders WHERE orders.user_id = users.id) AND status = 'active'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SELECT * FROM users WHERE age gt ? and EXISTS ((SELECT ? FROM orders WHERE orders.user_id eq users.id)) and status eq ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_Default(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test simple DEFAULT
	sql := "INSERT INTO users (name, created_at) VALUES ('Alice', DEFAULT)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, created_at) VALUES (?, DEFAULT)"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal("Alice", params[0][0])
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// Test DEFAULT with column name
	sql = "INSERT INTO users (name, age) VALUES (DEFAULT(name), DEFAULT(age))"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age) VALUES (DEFAULT name, DEFAULT age)"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// Test DEFAULT in multiple rows
	sql = "INSERT INTO users (name, age) VALUES ('Alice', 25), (DEFAULT, DEFAULT)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age) VALUES (?, ?), (DEFAULT, DEFAULT)"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal("Alice", params[0][0])
	as.Equal(int64(25), params[0][1])
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// Test DEFAULT with other expressions
	sql = "INSERT INTO users (name, age, created_at) VALUES (DEFAULT, 26, DEFAULT)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"INSERT INTO users (name, age, created_at) VALUES (DEFAULT, ?, DEFAULT)"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal(int64(26), params[0][0])
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)
}

func TestTemplatizeSQL_TimeUnit(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// DATE_SUB with DAY
	sql := "SELECT * FROM orders WHERE create_time > DATE_SUB(NOW(), INTERVAL 7 DAY)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT * FROM orders WHERE create_time gt DATE_SUB(NOW(), INTERVAL ? DAY)"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// DATE_ADD with HOUR
	sql = "SELECT * FROM events WHERE event_time < DATE_ADD(NOW(), INTERVAL 24 HOUR)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT * FROM events WHERE event_time lt DATE_ADD(NOW(), INTERVAL ? HOUR)"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "events", "", "events"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Multiple intervals in one query
	sql = "SELECT * FROM logs WHERE created_at BETWEEN DATE_SUB(NOW(), INTERVAL 30 DAY) AND DATE_SUB(NOW(), INTERVAL 1 DAY)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT * FROM logs WHERE created_at BETWEEN DATE_SUB(NOW(), INTERVAL ? DAY) AND DATE_SUB(NOW(), INTERVAL ? DAY)"},
		template)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "logs", "", "logs"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// UPDATE with DATE_ADD
	sql = "UPDATE tasks SET due_date = DATE_ADD(created_at, INTERVAL 30 MINUTE)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"UPDATE tasks SET due_date eq DATE_ADD(created_at, INTERVAL ? MINUTE)"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "tasks", "", "tasks"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// SELECT with alias and MONTH interval
	sql = "SELECT DATE_ADD(start_date, INTERVAL 3 MONTH) as end_date FROM projects"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT DATE_ADD(start_date, INTERVAL ? MONTH) AS end_date FROM projects"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "projects", "", "projects"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// "Complex query with multiple time functions"
	sql = "SELECT * FROM events WHERE start_time > DATE_SUB(NOW(), INTERVAL 1 DAY) AND end_time < DATE_ADD(NOW(), INTERVAL 7 DAY)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT * FROM events WHERE start_time gt DATE_SUB(NOW(), INTERVAL ? DAY) and end_time lt DATE_ADD(NOW(), INTERVAL ? DAY)"},
		template)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "events", "", "events"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// "YEAR interval with decimal"
	sql = "SELECT * FROM employees WHERE hire_date < DATE_SUB(NOW(), INTERVAL 2.5 YEAR)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal(
		[]string{"SELECT * FROM employees WHERE hire_date lt DATE_SUB(NOW(), INTERVAL ? YEAR)"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "employees", "", "employees"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_Explain(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test basic EXPLAIN
	sql := "EXPLAIN SELECT * FROM users WHERE id = 1"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"EXPLAIN FORMAT = row SELECT * FROM users WHERE id eq ?"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationExplain}, op)

	// Test EXPLAIN ANALYZE
	sql = "EXPLAIN ANALYZE SELECT * FROM users WHERE name = 'kyden' AND age > 18"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"EXPLAIN ANALYZE FORMAT = row SELECT * FROM users WHERE name eq ? and age gt ?"},
		template)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationExplain}, op)

	// Test EXPLAIN with FORMAT
	sql = "EXPLAIN FORMAT = JSON SELECT * FROM users WHERE id IN (1, 2, 3)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"EXPLAIN FORMAT = JSON SELECT * FROM users WHERE id IN (?, ?, ?)"},
		template)
	as.Equal(1, len(params))
	as.Equal(3, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationExplain}, op)

	// Test EXPLAIN ANALYZE with FORMAT
	sql = "EXPLAIN ANALYZE FORMAT = JSON SELECT u.* FROM users u JOIN orders o ON u.id = o.user_id WHERE o.status = 'pending'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"EXPLAIN ANALYZE FORMAT = JSON SELECT u.* FROM users AS u CROSS JOIN orders AS o ON u.id eq o.user_id WHERE o.status eq ?"},
		template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationExplain}, op)
}

func TestTemplatizeSQL_InvalidSQL(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// 测试语法错误的SQL
	sql := "SELECT * FROM users WHERE"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.NotNil(err)
	as.Equal([]string(nil), template)
	as.Equal(0, len(params))
	as.Equal(0, len(tableInfos))
	as.Equal([]models.SQLOpType(nil), op)

	// 测试空的SQL语句列表
	sql = ";"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal("no valid SQL statements found", err.Error())
	as.Equal([]string(nil), template)
	as.Equal(0, len(params))
	as.Equal(0, len(tableInfos))
	as.Equal([]models.SQLOpType(nil), op)
}

func TestTemplatizeSQL_CrossJoin(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// CROSS JOIN
	sql := "SELECT * FROM users CROSS JOIN orders"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Nil(err)
	as.Equal([]string{
		"SELECT * FROM users CROSS JOIN orders",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// INNER JOIN
	sql = "SELECT * FROM users INNER JOIN orders ON users.id = orders.user_id"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal([]string{
		"SELECT * FROM users CROSS JOIN orders ON users.id eq orders.user_id",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_RightJoin(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	sql := "SELECT * FROM users RIGHT JOIN orders ON users.id = orders.user_id"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Nil(err)
	as.Equal([]string{
		"SELECT * FROM users RIGHT JOIN orders ON users.id eq orders.user_id",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_UnaryOperations(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// 测试 NOT 操作符
	sql := "SELECT * FROM users WHERE NOT (age > 18)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Nil(err)
	as.Equal([]string{
		"SELECT * FROM users WHERE not (age gt ?)",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// 测试负数
	sql = "SELECT * FROM users WHERE balance < -100"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Nil(err)
	as.Equal([]string{
		"SELECT * FROM users WHERE balance lt minus ?",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_MultipleErrors(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// 测试语法错误
	sql := "SELECT * FROM users WHERE id = ;"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.NotNil(err)
	as.Equal([]string(nil), template)
	as.Equal(0, len(params))
	as.Equal([][]*models.TableInfo(nil), tableInfos)
	as.Equal([]models.SQLOpType(nil), op)

	// 测试不完整的SQL
	sql = "SELECT * FROM"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.NotNil(err)
	as.Equal([]string(nil), template)
	as.Equal(0, len(params))
	as.Equal([][]*models.TableInfo(nil), tableInfos)
	as.Equal([]models.SQLOpType(nil), op)
}

func TestTemplatizeSQL_SubqueryCompare(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test subquery with comparison operators
	sql := "SELECT * FROM users WHERE age > (SELECT AVG(age) FROM users) AND salary >= ANY(SELECT salary FROM managers)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT * FROM users WHERE age gt (SELECT AVG(age) FROM users) and salary ge ANY((SELECT salary FROM managers))",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
		models.NewTableInfo("", "managers", "", "managers"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test subquery with ALL
	sql = "SELECT * FROM employees WHERE salary > ALL(SELECT salary FROM interns WHERE department = 'IT')"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT * FROM employees WHERE salary gt ALL((SELECT salary FROM interns WHERE department eq ?))",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "employees", "", "employees"),
		models.NewTableInfo("", "interns", "", "interns"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_NestedFunctions(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test nested function calls
	sql := "SELECT DATE_FORMAT(FROM_UNIXTIME(create_time), '%Y-%m-%d') as date FROM orders"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT DATE_FORMAT(FROM_UNIXTIME(create_time), ?) AS date FROM orders",
	}, template)
	as.Equal(1, len(params))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test function with subquery
	sql = "SELECT COALESCE((SELECT name FROM users WHERE id = 1), 'Unknown') as username"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT COALESCE((SELECT name FROM users WHERE id eq ?), ?) AS username",
	}, template)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "users", "", "users"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_ComplexConditions(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test complex WHERE conditions with multiple operators
	sql := "SELECT * FROM products WHERE (price BETWEEN 100 AND 200 OR stock > 0) AND (category IN ('electronics', 'books') OR name LIKE '%special%')"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT * FROM products WHERE (price BETWEEN ? AND ? or stock gt ?) and (category IN (?, ?) or name LIKE ?)",
	}, template)
	as.Equal(1, len(params))
	as.Equal(6, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "products", "", "products"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// Test complex conditions with NULL checks
	sql = "SELECT * FROM orders WHERE status IS NOT NULL AND (total > 1000 OR customer_id IN (SELECT id FROM vip_customers))"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT * FROM orders WHERE status IS NOT NULL and (total gt ? or customer_id IN ((SELECT id FROM vip_customers)))",
	}, template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
		models.NewTableInfo("", "vip_customers", "", "vip_customers"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeSQL_TimeUnitExpr(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test time unit expression
	sql := "SELECT * FROM orders WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 DAY)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT * FROM orders WHERE created_at gt DATE_SUB(NOW(), INTERVAL ? DAY)",
	}, template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
}

func TestTemplatizeVisitor_logError(t *testing.T) {
	t.Parallel()

	v := &ExtractVisitor{}
	v.logError("test")
}

func TestTemplatizeSQL_EmptySpace(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// select
	sql := "  SELECT * FROM orders WHERE created_at >  DATE_SUB(NOW(), INTERVAL 1 DAY)"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"SELECT * FROM orders WHERE created_at gt DATE_SUB(NOW(), INTERVAL ? DAY)",
	}, template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)

	// insert
	sql = "  INSERT INTO orders (created_at, total) VALUES (DATE_SUB(NOW(), INTERVAL 1 DAY), 100)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"INSERT INTO orders (created_at, total) VALUES (DATE_SUB(NOW(), INTERVAL ? DAY), ?)",
	}, template)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)

	// update
	sql = "  UPDATE orders SET total = total - 100 WHERE created_at >  DATE_SUB(NOW(), INTERVAL 1 DAY)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"UPDATE orders SET total eq total minus ? WHERE created_at gt DATE_SUB(NOW(), INTERVAL ? DAY)",
	}, template)
	as.Equal(1, len(params))
	as.Equal(2, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)

	// delete
	sql = "  DELETE FROM orders WHERE created_at >  DATE_SUB(NOW(), INTERVAL 1 DAY)"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal([]string{
		"DELETE FROM orders WHERE created_at gt DATE_SUB(NOW(), INTERVAL ? DAY)",
	}, template)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal([][]*models.TableInfo{{
		models.NewTableInfo("", "orders", "", "orders"),
	}}, tableInfos)
	as.Equal([]models.SQLOpType{models.SQLOperationDelete}, op)
}

func TestTemplatizeSQL_ShowStatements(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	parser := NewExtractor()

	// Test SHOW CREATE TABLE
	sql := "SHOW CREATE TABLE `tbUserTask_6`"
	template, tableInfos, params, op, err := parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW CREATE TABLE tbUserTask_6"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0])) // 应该没有表信息
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW CREATE DATABASE
	sql = "SHOW CREATE DATABASE test_db"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW CREATE DATABASE test_db"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW CREATE DATABASE IF NOT EXISTS
	sql = "SHOW CREATE DATABASE IF NOT EXISTS test_db"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW CREATE DATABASE test_db IF NOT EXISTS"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW DATABASES
	sql = "SHOW DATABASES"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW DATABASES"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW DATABASES LIKE
	sql = "SHOW DATABASES LIKE 'test%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW DATABASES LIKE ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal("test%", params[0][0])
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW TABLES
	sql = "SHOW TABLES"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW TABLES"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW TABLES FROM
	sql = "SHOW TABLES FROM test_db"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW TABLES FROM test_db"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW TABLES LIKE
	sql = "SHOW TABLES LIKE 'user%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW TABLES LIKE ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal("user%", params[0][0])
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW TABLES WHERE
	sql = "SHOW TABLES WHERE `Table_type` = 'BASE TABLE'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW TABLES WHERE Table_type eq ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal("BASE TABLE", params[0][0])
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW COLUMNS
	sql = "SHOW COLUMNS FROM users"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW COLUMNS FROM users"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW COLUMNS FROM schema.table
	sql = "SHOW COLUMNS FROM mydb.users"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW COLUMNS FROM mydb.users"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW COLUMNS LIKE
	sql = "SHOW COLUMNS FROM users LIKE 'id%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW COLUMNS FROM users LIKE ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal("id%", params[0][0])
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW INDEX
	sql = "SHOW INDEX FROM users"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW INDEX FROM users"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW PROCESSLIST
	sql = "SHOW PROCESSLIST"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW PROCESSLIST"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW FULL PROCESSLIST
	sql = "SHOW FULL PROCESSLIST"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW FULL PROCESSLIST"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW VARIABLES
	sql = "SHOW VARIABLES"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW VARIABLES"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW VARIABLES LIKE
	sql = "SHOW VARIABLES LIKE 'max_%'"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW VARIABLES LIKE ?"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(1, len(params[0]))
	as.Equal("max_%", params[0][0])
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW STATUS
	sql = "SHOW STATUS"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW STATUS"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW TABLE STATUS
	sql = "SHOW TABLE STATUS"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW TABLE STATUS"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW TABLE STATUS FROM
	sql = "SHOW TABLE STATUS FROM test_db"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW TABLE STATUS FROM test_db"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW WARNINGS
	sql = "SHOW WARNINGS"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW WARNINGS"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)

	// Test SHOW ERRORS
	sql = "SHOW ERRORS"
	template, tableInfos, params, op, err = parser.Extract(sql)
	as.Equal(nil, err)
	as.Equal(
		[]string{"SHOW ERRORS"},
		template,
	)
	as.Equal(1, len(params))
	as.Equal(0, len(params[0]))
	as.Equal(1, len(tableInfos))
	as.Equal(0, len(tableInfos[0]))
	as.Equal([]models.SQLOpType{models.SQLOperationShow}, op)
}

func TestExtractor_EscapedQuotes(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extractor := NewExtractor()

	// Test SQL with escaped single quotes
	sql := "select * from tbGameCoinSerialV2 where   `iStatus` != 0 and `dtCommitTime` < '2025-06-10 13:40:00'  order by `iSeqId` asc limit 5000"
	template, tableInfos, params, op, err := extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM tbGameCoinSerialV2 WHERE iStatus ne ? and dtCommitTime lt ? ORDER BY iSeqId LIMIT ?"},
		template,
	)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "tbGameCoinSerialV2", "", "tbGameCoinSerialV2")}}, tableInfos)

	// Test SQL with mixed quotes (both escaped and regular)
	sql = "SELECT * FROM users WHERE name = 'normal' AND created_at < '2025-06-10'"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ? and created_at lt ?"},
		template,
	)
	as.Equal([][]any{{"normal", "2025-06-10"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users", "", "users")}}, tableInfos)
}

func TestExtractor_AdvancedPreprocessing(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extractor := NewExtractor()

	// Test SQL with escaped double quotes
	sql := "SELECT * FROM products WHERE description LIKE 'Premium quality'"
	template, tableInfos, params, op, err := extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM products WHERE description LIKE ?"},
		template,
	)
	as.Equal([][]any{{"Premium quality"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "products", "", "products")}}, tableInfos)

	// Test SQL with double backslashes
	sql = "SELECT * FROM files WHERE path = 'C:\\Windows\\System32'"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM files WHERE path eq ?"},
		template,
	)
	as.Equal([][]any{{"C:WindowsSystem32"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "files", "", "files")}}, tableInfos)

	// Test SQL with Unicode escape sequences
	sql = "SELECT * FROM users WHERE name LIKE '中文'"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name LIKE ?"},
		template,
	)
	as.Equal([][]any{{"中文"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users", "", "users")}}, tableInfos)

	// Test SQL with null bytes (which could be malicious)
	sql = "SELECT * FROM users WHERE username = 'admin' OR 1=1"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM users WHERE username eq ? or ? eq ?"},
		template,
	)
	as.Equal([][]any{{"admin", int64(1), int64(1)}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users", "", "users")}}, tableInfos)

	// Test SQL with extra whitespace
	sql = "  SELECT   *   FROM   users   WHERE   name   =   'John'   "
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM users WHERE name eq ?"},
		template,
	)
	as.Equal([][]any{{"John"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users", "", "users")}}, tableInfos)

	// Test SQL with complex date format and escaped quotes
	sql = "SELECT * FROM orders WHERE created_at BETWEEN '2025-01-01 00:00:00' AND '2025-12-31 23:59:59'"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM orders WHERE created_at BETWEEN ? AND ?"},
		template,
	)
	as.Equal([][]any{{"2025-01-01 00:00:00", "2025-12-31 23:59:59"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "orders", "", "orders")}}, tableInfos)

	// Test with quoted identifiers
	sql = "SELECT `id`, `name` FROM `users` WHERE `status` = 'active'"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT id, name FROM users WHERE status eq ?"},
		template,
	)
	as.Equal([][]any{{"active"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "users", "", "users")}}, tableInfos)
}

func TestExtractor_ComplexEscapeSequences(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	extractor := NewExtractor()

	// Test SQL with mixed escaped quotes and special characters
	sql := "SELECT * FROM logs WHERE message LIKE '%Error at line %' AND timestamp > '2025-01-01'"
	template, tableInfos, params, op, err := extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM logs WHERE message LIKE ? and timestamp gt ?"},
		template,
	)
	as.Equal([][]any{{"%Error at line %", "2025-01-01"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "logs", "", "logs")}}, tableInfos)

	// Test SQL with escaped quotes in multiple places
	sql = "UPDATE products SET description = 'Product with special features' WHERE id = 1"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationUpdate}, op)
	as.Equal(
		[]string{"UPDATE products SET description eq ? WHERE id eq ?"},
		template,
	)
	as.Equal([][]any{{"Product with special features", int64(1)}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "products", "", "products")}}, tableInfos)

	// Test SQL with multiple escaped sequences
	sql = "INSERT INTO events (name, description) VALUES ('New Years Eve', 'Celebration on Dec 31st')"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationInsert}, op)
	as.Equal(
		[]string{"INSERT INTO events (name, description) VALUES (?, ?)"},
		template,
	)
	as.Equal([][]any{{"New Years Eve", "Celebration on Dec 31st"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "events", "", "events")}}, tableInfos)

	// Test SQL with both single and double quotes
	sql = "SELECT * FROM products WHERE name = 'Mens Premium Shirt'"
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT * FROM products WHERE name eq ?"},
		template,
	)
	as.Equal([][]any{{"Mens Premium Shirt"}}, params)
	as.Equal([][]*models.TableInfo{{models.NewTableInfo("", "products", "", "products")}}, tableInfos)

	// Test SQL with complex nested conditions and quotes
	sql = `
		SELECT p.*, c.name as category_name 
		FROM products p 
		JOIN categories c ON p.category_id = c.id 
		WHERE (p.price > 100 AND p.stock > 0) 
		OR (p.name LIKE '%Limited Edition%' AND p.release_date > '2025-01-01')
		ORDER BY p.price DESC
		LIMIT 10
	`
	template, tableInfos, params, op, err = extractor.Extract(sql)
	as.Nil(err)
	as.Equal([]models.SQLOpType{models.SQLOperationSelect}, op)
	as.Equal(
		[]string{"SELECT p.*, c.name AS category_name FROM products AS p CROSS JOIN categories AS c ON p.category_id eq c.id WHERE (p.price gt ? and p.stock gt ?) or (p.name LIKE ? and p.release_date gt ?) ORDER BY p.price DESC LIMIT ?"},
		template,
	)
	as.Equal([][]any{{int64(100), int64(0), "%Limited Edition%", "2025-01-01", uint64(10)}}, params)
}

func TestTemplateTable(t *testing.T) {
	t.Parallel()
	as := assert.New(t)
	visitor := &ExtractVisitor{}

	testCases := []struct {
		name           string
		inputSchema    string
		inputTable     string
		expectedSchema string
		expectedTable  string
	}{
		{
			name:           "empty schema and table",
			inputSchema:    "",
			inputTable:     "",
			expectedSchema: "",
			expectedTable:  "",
		},
		{
			name:           "normal schema and table without underscore",
			inputSchema:    "db",
			inputTable:     "users",
			expectedSchema: "db",
			expectedTable:  "users",
		},
		{
			name:           "schema with underscore but no number",
			inputSchema:    "my_db",
			inputTable:     "users",
			expectedSchema: "my_db",
			expectedTable:  "users",
		},
		{
			name:           "table with underscore but no number",
			inputSchema:    "db",
			inputTable:     "user_info",
			expectedSchema: "db",
			expectedTable:  "user_info",
		},
		{
			name:           "schema with underscore and number",
			inputSchema:    "db_123",
			inputTable:     "users",
			expectedSchema: "db_?",
			expectedTable:  "users",
		},
		{
			name:           "table with underscore and number",
			inputSchema:    "db",
			inputTable:     "users_4",
			expectedSchema: "db",
			expectedTable:  "users_?",
		},
		{
			name:           "both schema and table with underscore and number",
			inputSchema:    "db_123",
			inputTable:     "users_45",
			expectedSchema: "db_?",
			expectedTable:  "users_?",
		},
		{
			name:           "multiple underscores in schema",
			inputSchema:    "my_db_123",
			inputTable:     "users",
			expectedSchema: "my_db_?",
			expectedTable:  "users",
		},
		{
			name:           "multiple underscores in table",
			inputSchema:    "db",
			inputTable:     "user_info_789",
			expectedSchema: "db",
			expectedTable:  "user_info_?",
		},
		{
			name:           "multiple underscores in both",
			inputSchema:    "my_db_123",
			inputTable:     "user_info_789",
			expectedSchema: "my_db_?",
			expectedTable:  "user_info_?",
		},
		{
			name:           "underscore with non-numeric suffix",
			inputSchema:    "db_test",
			inputTable:     "users_prod",
			expectedSchema: "db_test",
			expectedTable:  "users_prod",
		},
		{
			name:           "complex sharding pattern",
			inputSchema:    "db_shard_001",
			inputTable:     "users_region_002",
			expectedSchema: "db_shard_?",
			expectedTable:  "users_region_?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			table := visitor.templateTable(tc.inputTable)
			as.Equal(tc.expectedTable, table)

			schema := visitor.templateTable(tc.inputSchema)
			as.Equal(tc.expectedSchema, schema)
		})
	}
}
