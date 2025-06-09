package extract

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/kydance/ziwi/slices"
	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/test_driver"

	"github.com/kydance/sql-extractor/internal/models"
)

const paramsMaxCount = 64

type Extractor struct {
	parser *parser.Parser

	pool sync.Pool
}

func NewExtractor() *Extractor {
	return &Extractor{
		parser: parser.New(),
		pool: sync.Pool{
			New: func() any {
				return &ExtractVisitor{
					builder:    &strings.Builder{},
					params:     make([]any, 0, paramsMaxCount),
					tableInfos: make([]*models.TableInfo, 0, paramsMaxCount),
					opType:     models.SQLOperationUnknown,
				}
			},
		},
	}
}

// Extract returns the templatized SQL, table info, parameters and operation type.
// It supports multiple SQL statements separated by semicolons.
func (p *Extractor) Extract(sql string) (
	[]string, [][]*models.TableInfo, [][]any, []models.SQLOpType, error,
) {
	if sql == "" {
		return nil, nil, nil, nil, errors.New("empty SQL statement")
	}

	stmts, _, err := p.parser.Parse(sql, "", "")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if len(stmts) == 0 {
		return nil, nil, nil, nil, errors.New("no valid SQL statements found")
	}

	// Handle multiple statements
	var (
		allTemplatizedSQL = make([]string, 0, len(stmts))
		allParams         = make([][]any, 0, len(stmts))
		allTableInfos     = make([][]*models.TableInfo, 0, len(stmts))
		opType            = make([]models.SQLOpType, 0, len(stmts))
	)

	for idx := range stmts {
		templatedSQL, tableInfos, params, op, err := p.extractOneStmt(stmts[idx])
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("error processing statement %d: %w", idx+1, err)
		}

		allTemplatizedSQL = append(allTemplatizedSQL, templatedSQL)
		allParams = append(allParams, params)
		allTableInfos = append(allTableInfos, tableInfos)
		opType = append(opType, op)
	}

	return allTemplatizedSQL, allTableInfos, allParams, opType, nil
}

// extractOneStmt handles a single SQL statement
func (p *Extractor) extractOneStmt(stmt ast.StmtNode) (
	string, []*models.TableInfo, []any, models.SQLOpType, error,
) {
	v, ok := p.pool.Get().(*ExtractVisitor)
	if !ok {
		return "", nil, nil, models.SQLOperationUnknown,
			errors.New("failed to get ExtractVisitor from pool")
	}

	defer func() {
		v.builder.Reset()
		v.params = v.params[:0]
		v.tableInfos = v.tableInfos[:0]
		v.inAggrFunc = false
		v.opType = models.SQLOperationUnknown

		p.pool.Put(v)
	}()

	stmt.Accept(v)

	return v.builder.String(),
		slices.UniqBy(v.tableInfos, func(t *models.TableInfo) string {
			if t.Schema() == "" {
				return t.TableName()
			}

			return t.Schema() + "." + t.TableName()
		}),
		v.params,
		v.opType,
		nil
}

// ExtractVisitor 实现 ast.Visitor 接口
type ExtractVisitor struct {
	builder    *strings.Builder
	params     []any
	inAggrFunc bool
	tableInfos []*models.TableInfo
	opType     models.SQLOpType
}

// 避免重复字符串操作
var joinTypeMap = map[ast.JoinType]string{
	ast.LeftJoin:  " LEFT JOIN ",
	ast.RightJoin: " RIGHT JOIN ",
	ast.CrossJoin: " CROSS JOIN ",
}

// Enter implement ast.Visitor interface. It handles ast.Node
//
// Return: nil, true - 不继续遍历， n, false - 继续遍历
//
//nolint:gocyclo,cyclop
func (v *ExtractVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if n == nil {
		return n, false
	}

	switch node := n.(type) {
	// 1. 基础表达式层 - 最常用的表达式处理
	case *ast.ColumnNameExpr:
		v.handleColumnNameExpr(node)
	case *test_driver.ValueExpr:
		v.handleValueExpr(node)
	case *ast.BinaryOperationExpr: // e.g 1+1, and
		v.handleBinaryOperationExpr(node)
	case *ast.TableName:
		v.handleTableName(node)

	// 2. SQL 语句层
	case *ast.SelectStmt:
		v.handleSelectStmt(node)
	case *ast.InsertStmt:
		v.handleInsertStmt(node)
	case *ast.UpdateStmt:
		v.handleUpdateStmt(node)
	case *ast.DeleteStmt:
		v.handleDeleteStmt(node)
	case *ast.ExplainStmt:
		v.handleExplainStmt(node)
	case *ast.ShowStmt:
		v.handleShowStmt(node)

	// 3. 表结构层 - 表引用和连接
	case *ast.TableSource:
		v.handleTableSource(node)
	case *ast.Join:
		v.handleJoin(node)
	case *ast.OnCondition:
		v.handleOnCondition(node)

	// 4. 条件表达式层 - WHERE/HAVING 子句中的条件
	case *ast.PatternInExpr:
		v.handlePatternInExpr(node)
	case *ast.PatternLikeOrIlikeExpr:
		v.handlePatternLikeOrIlikeExpr(node)
	case *ast.BetweenExpr:
		v.handleBetweenExpr(node)
	case *ast.ParenthesesExpr:
		v.handleParenthesesExpr(node)
	case *ast.CaseExpr:
		v.handleCaseExpr(node)
	case *ast.CompareSubqueryExpr:
		v.handleCompareSubqueryExpr(node)

	// 5. 函数和聚合层
	case *ast.FuncCallExpr:
		v.handleFuncCallExpr(node)
	case *ast.AggregateFuncExpr:
		old := v.inAggrFunc
		v.inAggrFunc = true
		defer func() { v.inAggrFunc = old }()
		v.handleAggregateFuncExpr(node)
	case *ast.UnaryOperationExpr:
		v.handleUnaryOperationExpr(node)
	case *ast.TimeUnitExpr:
		v.handleTimeUnitExpr(node)

	// 6. 修饰语层 - ORDER BY, LIMIT 等
	case *ast.ByItem:
		v.handleByItem(node)
	case *ast.Limit:
		v.handleLimit(node)
	case *ast.Assignment:
		v.handleAssignment(node)
	case *ast.ValuesExpr:
		v.handleValuesExpr(node)

	// 7. 子查询层 - 最复杂的查询结构
	case *ast.SubqueryExpr:
		v.handleSubqueryExpr(node)
	case *ast.IsNullExpr:
		v.handleIsNullExpr(node)
	case *ast.ExistsSubqueryExpr:
		v.handleExistsSubqueryExpr(node)

	// 8. 处理 DEFAULT 表达式
	case *ast.DefaultExpr:
		v.handleDefaultExpr(node)

	default:
		// FIXME IsTruthExpr
		// FIXME PatternRegexpExpr
		// FIXME PositionExpr
		// FIXME RowExpr
		// FIXME VariableExpr
		// FIXME MatchAgainst
		// FIXME SetCollationExpr
		v.logError(fmt.Sprintf("Enter ast.Node type: %T", node))
	}

	return n, true
}

// Leave 实现 ast.Visitor 接口.
// Return: n, true - 不继续遍历
func (v *ExtractVisitor) Leave(n ast.Node) (ast.Node, bool) {
	return n, true
}

// SELECT 子句: SELECT 列表、FROM 子句、WHERE 子句、GROUP BY 子句、HAVING 子句、ORDER BY 子句、LIMIT 子句
//
// nolint:cyclop
func (v *ExtractVisitor) handleSelectStmt(node *ast.SelectStmt) {
	if v.opType == models.SQLOperationUnknown {
		v.opType = models.SQLOperationSelect
	}

	v.builder.WriteString("SELECT ")

	// DISTINCT 关键字
	if node.Distinct {
		v.builder.WriteString("DISTINCT ")
	}

	// 处理 SELECT 列表
	if node.Fields != nil {
		for idx := range node.Fields.Fields {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			if node.Fields.Fields[idx].WildCard != nil { // *
				// Schema
				if node.Fields.Fields[idx].WildCard.Schema.O != "" {
					v.builder.WriteString(node.Fields.Fields[idx].WildCard.Schema.O)
					v.builder.WriteString(".")
				}

				if node.Fields.Fields[idx].WildCard.Table.O != "" {
					v.builder.WriteString(node.Fields.Fields[idx].WildCard.Table.O)
					v.builder.WriteString(".")
				}

				v.builder.WriteString("*")
			} else {
				node.Fields.Fields[idx].Expr.Accept(v)

				// 处理 AS
				if node.Fields.Fields[idx].AsName.String() != "" {
					v.builder.WriteString(" AS ")
					v.builder.WriteString(node.Fields.Fields[idx].AsName.String())
				}
			}
		}
	}

	// FROM 子句
	if node.From != nil {
		v.builder.WriteString(" FROM ")
		if node.From.TableRefs != nil {
			node.From.TableRefs.Accept(v)
		}
	}

	// WHERE 子句
	if node.Where != nil {
		v.builder.WriteString(" WHERE ")
		node.Where.Accept(v)
	}

	// GROUP BY 子句
	if node.GroupBy != nil {
		v.builder.WriteString(" GROUP BY ")
		for idx, item := range node.GroupBy.Items {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			item.Accept(v)
		}
	}

	// HAVING 子句
	if node.Having != nil && node.Having.Expr != nil {
		v.builder.WriteString(" HAVING ")

		switch expr := node.Having.Expr.(type) {
		case *ast.BinaryOperationExpr:
			expr.Accept(v)

		default:
			v.logError(fmt.Sprintf("Having.Expr type: %T", expr))
			expr.Accept(v)
		}
	}

	// ORDER BY 子句
	if node.OrderBy != nil {
		v.builder.WriteString(" ORDER BY ")
		for idx, item := range node.OrderBy.Items {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			item.Accept(v)
		}
	}

	// LIMIT 子句
	if node.Limit != nil {
		node.Limit.Accept(v)
	}
}

// INSERT 语句
func (v *ExtractVisitor) handleInsertStmt(node *ast.InsertStmt) {
	if v.opType == models.SQLOperationUnknown {
		v.opType = models.SQLOperationInsert
	}

	v.builder.WriteString("INSERT ")
	// INSERT IGNORE
	if node.IgnoreErr {
		v.builder.WriteString("IGNORE ")
	}
	v.builder.WriteString("INTO ")

	// TABLE
	if node.Table.TableRefs != nil {
		node.Table.TableRefs.Accept(v) // call handleTableSource()
	}

	// COLUMNS
	if len(node.Columns) > 0 {
		v.builder.WriteString(" (")
		for idx, col := range node.Columns {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			v.builder.WriteString(col.Name.O)
		}
		v.builder.WriteString(")")
	}

	// VALUES
	if node.Lists != nil {
		v.builder.WriteString(" VALUES ")
		for idx, list := range node.Lists {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			v.builder.WriteString("(")
			for jdx, item := range list {
				if jdx > 0 {
					v.builder.WriteString(", ")
				}

				item.Accept(v)
			}
			v.builder.WriteString(")")
		}
	} else if node.Select != nil { // INSERT ... SELECT ...
		v.builder.WriteString(" ")
		node.Select.Accept(v)
	}

	// ON DUPLICATE KEY UPDATE
	if node.OnDuplicate != nil {
		v.builder.WriteString(" ON DUPLICATE KEY UPDATE ")

		for idx := range node.OnDuplicate {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			node.OnDuplicate[idx].Accept(v)
		}
	}
}

// UPDATE
func (v *ExtractVisitor) handleUpdateStmt(node *ast.UpdateStmt) {
	if v.opType == models.SQLOperationUnknown {
		v.opType = models.SQLOperationUpdate
	}

	v.builder.WriteString("UPDATE ")

	if node.TableRefs != nil && node.TableRefs.TableRefs != nil {
		node.TableRefs.TableRefs.Accept(v) // call handleTableSource()
	}

	// SET
	v.builder.WriteString(" SET ")
	for idx := range node.List {
		if idx > 0 {
			v.builder.WriteString(", ")
		}

		node.List[idx].Accept(v)
	}

	// WHERE
	if node.Where != nil {
		v.builder.WriteString(" WHERE ")
		node.Where.Accept(v)
	}

	// ORDER BY
	if node.Order != nil {
		v.builder.WriteString(" ORDER BY ")
		for idx := range node.Order.Items {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			node.Order.Items[idx].Accept(v)
		}
	}

	// LIMIT
	if node.Limit != nil {
		node.Limit.Accept(v)
	}
}

// DELETE
func (v *ExtractVisitor) handleDeleteStmt(node *ast.DeleteStmt) {
	if v.opType == models.SQLOperationUnknown {
		v.opType = models.SQLOperationDelete
	}

	v.builder.WriteString("DELETE ")

	if node.Tables != nil {
		for idx := range node.Tables.Tables {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			node.Tables.Tables[idx].Accept(v)
		}
		v.builder.WriteString(" ")
	}
	v.builder.WriteString("FROM ")

	// TABLE
	if node.TableRefs != nil && node.TableRefs.TableRefs != nil { // ast.Join
		node.TableRefs.TableRefs.Accept(v)
	}

	// WHERE
	if node.Where != nil {
		v.builder.WriteString(" WHERE ")
		node.Where.Accept(v)
	}

	// ORDER BY
	if node.Order != nil {
		v.builder.WriteString(" ORDER BY ")
		for idx := range node.Order.Items {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			node.Order.Items[idx].Accept(v)
		}
	}

	// LIMIT
	if node.Limit != nil {
		node.Limit.Accept(v)
	}
}

// handleExplainStmt 处理 EXPLAIN 语句
func (v *ExtractVisitor) handleExplainStmt(node *ast.ExplainStmt) {
	if v.opType == models.SQLOperationUnknown {
		v.opType = models.SQLOperationExplain
	}

	v.builder.WriteString("EXPLAIN ")
	if node.Analyze {
		v.builder.WriteString("ANALYZE ")
	}
	if node.Format != "" {
		v.builder.WriteString("FORMAT = ")
		v.builder.WriteString(node.Format)
		v.builder.WriteString(" ")
	}

	// 递归处理被解释的语句
	if node.Stmt != nil {
		node.Stmt.Accept(v)
	}
}

// handleTableSource 处理表源
func (v *ExtractVisitor) handleTableSource(node *ast.TableSource) {
	switch src := node.Source.(type) {
	case *ast.TableName:
		src.Accept(v)

	case *ast.SelectStmt:
		v.builder.WriteString("(")
		src.Accept(v)
		v.builder.WriteString(")")

	case *ast.Join:
		src.Accept(v)

	default:
		v.logError(fmt.Sprintf("TableSource.Source type: %T", src))
		node.Source.Accept(v)
	}

	if node.AsName.O != "" {
		v.builder.WriteString(" AS ")
		v.builder.WriteString(node.AsName.O)
	}
}

func (v *ExtractVisitor) handleTableName(node *ast.TableName) {
	v.tableInfos = append(v.tableInfos, models.NewTableInfo())

	if node.Schema.O != "" {
		v.builder.WriteString(node.Schema.O)
		v.builder.WriteString(".")

		v.tableInfos[len(v.tableInfos)-1].SetSchema(node.Schema.O)
	}

	v.builder.WriteString(node.Name.O)
	v.tableInfos[len(v.tableInfos)-1].SetTableName(node.Name.O)
}

func (v *ExtractVisitor) handleJoin(node *ast.Join) {
	if node.Left != nil {
		switch left := node.Left.(type) {
		// 若左节点是 JOIN，递归处理
		case *ast.Join:
			left.Accept(v)

		case *ast.TableSource:
			left.Accept(v)

		default:
			v.logError(fmt.Sprintf("Join.Left type: %T", left))
			left.Accept(v)
		}
	}

	// 只有存在右节点时，才添加 JOIN 关键字
	if node.Right != nil {
		// JOIN Type
		if joinStr, ok := joinTypeMap[node.Tp]; ok {
			v.builder.WriteString(joinStr)
		} else {
			v.builder.WriteString(" JOIN ")
		}

		switch right := node.Right.(type) {
		case *ast.TableSource:
			right.Accept(v)

		default:
			v.logError(fmt.Sprintf("Join.Right type: %T", right))
			node.Right.Accept(v)
		}

		// ON condition
		if node.On != nil {
			v.builder.WriteString(" ON ")
			node.On.Accept(v)
		}
	}
}

func (v *ExtractVisitor) handlePatternLikeOrIlikeExpr(node *ast.PatternLikeOrIlikeExpr) {
	node.Expr.Accept(v)
	if node.Not {
		v.builder.WriteString(" NOT")
	}
	v.builder.WriteString(" LIKE ")

	// 处理 LIKE 模式
	if pattern, ok := node.Pattern.(*test_driver.ValueExpr); ok {
		v.builder.WriteString("?")
		v.params = append(v.params, pattern.GetValue())
	} else {
		node.Pattern.Accept(v)
	}

	// FIXME 处理 LIKE 模式中的转义字符
	// if node.Escape != 0 {
	// 	v.builder.WriteString(" ESCAPE ")
	// 	v.builder.WriteString("?")
	// 	v.params = append(v.params, node.Escape)
	// }
}

func (v *ExtractVisitor) handlePatternInExpr(node *ast.PatternInExpr) {
	node.Expr.Accept(v)
	if node.Not {
		v.builder.WriteString(" NOT")
	}
	v.builder.WriteString(" IN (")

	if node.List != nil {
		for idx := range node.List {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			v.builder.WriteString("?")
			// 如果是 ValueExpr，保存参数值
			if valExpr, ok := node.List[idx].(*test_driver.ValueExpr); ok {
				v.params = append(v.params, valExpr.GetValue())
			}
		}
	}

	if node.Sel != nil {
		node.Sel.Accept(v)
	}

	v.builder.WriteString(")")
}

func (v *ExtractVisitor) handleBinaryOperationExpr(node *ast.BinaryOperationExpr) {
	node.L.Accept(v)
	fmt.Fprintf(v.builder, " %s ", node.Op.String())
	node.R.Accept(v)
}

func (v *ExtractVisitor) handleBetweenExpr(node *ast.BetweenExpr) {
	node.Expr.Accept(v)

	if node.Not {
		v.builder.WriteString("NOT ")
	}

	v.builder.WriteString(" BETWEEN ")
	node.Left.Accept(v)
	v.builder.WriteString(" AND ")
	node.Right.Accept(v)
}

func (v *ExtractVisitor) handleValueExpr(node *test_driver.ValueExpr) {
	if v.inAggrFunc { // 在聚合函数中，直接输出值
		switch val := node.GetValue().(type) {
		case int64, uint64:
			fmt.Fprintf(v.builder, "%d", val)

		case float64:
			fmt.Fprintf(v.builder, "%f", val)

		case string:
			fmt.Fprintf(v.builder, "'%s'", val)

		case *test_driver.MyDecimal:
			v.builder.WriteString(val.String())

		default:
			fmt.Printf("ValueExpr type: %T\n", node.GetValue())
			fmt.Fprintf(v.builder, "%v", val)
		}
	} else {
		// param -> ?
		v.builder.WriteString("?")
		v.params = append(v.params, node.GetValue())
	}
}

func (v *ExtractVisitor) handleColumnNameExpr(node *ast.ColumnNameExpr) {
	var schema, table string
	if node.Name.Schema.O != "" {
		schema = node.Name.Schema.O + "."
	}

	if node.Name.Table.O != "" {
		table = node.Name.Table.O + "."
	}

	v.builder.WriteString(schema + table + node.Name.Name.O)
}

func (v *ExtractVisitor) handleByItem(node *ast.ByItem) {
	node.Expr.Accept(v)

	// 处理排序方向
	if node.Desc {
		v.builder.WriteString(" DESC")
	}

	// FIXME 处理 NULL 排序
}

func (v *ExtractVisitor) handleValuesExpr(node *ast.ValuesExpr) {
	v.builder.WriteString("VALUES(")
	node.Column.Accept(v)
	// node.Accept(v)
	v.builder.WriteString(")")
}

func (v *ExtractVisitor) handleLimit(node *ast.Limit) {
	v.builder.WriteString(" LIMIT ")

	if node.Offset != nil {
		node.Offset.Accept(v)
		v.builder.WriteString(", ")
	}

	node.Count.Accept(v)
}

func (v *ExtractVisitor) handleSubqueryExpr(node *ast.SubqueryExpr) {
	v.builder.WriteString("(")
	node.Query.Accept(v)
	v.builder.WriteString(")")
}

func (v *ExtractVisitor) handleOnCondition(node *ast.OnCondition) {
	node.Expr.Accept(v)
}

// handleAssignment 处理赋值表达式
func (v *ExtractVisitor) handleAssignment(node *ast.Assignment) {
	v.handleColumnNameExpr(&ast.ColumnNameExpr{Name: node.Column}) // XXX
	v.builder.WriteString(" eq ")
	node.Expr.Accept(v)
}

// handleExprNode 处理表达式节点
func (v *ExtractVisitor) handleAggregateFuncExpr(node *ast.AggregateFuncExpr) {
	v.builder.WriteString(node.F)
	v.builder.WriteString("(")

	if node.Distinct {
		v.builder.WriteString("DISTINCT ")
	}

	for idx := range node.Args {
		if idx > 0 {
			v.builder.WriteString(", ")
		}

		node.Args[idx].Accept(v)
	}
	v.builder.WriteString(")")
}

// handleCaseExpr 处理 CASE 表达式
func (v *ExtractVisitor) handleCaseExpr(node *ast.CaseExpr) {
	if node == nil {
		return
	}

	v.builder.WriteString("CASE")

	// Simple CASE: CASE expr WHEN v1 THEN r1 [WHEN v2 THEN r2] [ELSE rn] END
	if node.Value != nil {
		v.builder.WriteString(" ")
		node.Value.Accept(v)
	}

	// Handle WHEN ... THEN clauses
	for idx := range node.WhenClauses {
		v.builder.WriteString(" WHEN ")
		node.WhenClauses[idx].Expr.Accept(v)
		v.builder.WriteString(" THEN ")
		node.WhenClauses[idx].Result.Accept(v)
	}

	// Handle ELSE clause
	if node.ElseClause != nil {
		v.builder.WriteString(" ELSE ")
		node.ElseClause.Accept(v)
	}

	v.builder.WriteString(" END")
}

// handleParenthesesExpr 处理括号表达式
func (v *ExtractVisitor) handleParenthesesExpr(node *ast.ParenthesesExpr) {
	v.builder.WriteString("(")
	node.Expr.Accept(v)
	v.builder.WriteString(")")
}

// handleFuncCallExpr 处理函数调用表达式
func (v *ExtractVisitor) handleFuncCallExpr(node *ast.FuncCallExpr) {
	v.builder.WriteString(node.FnName.String())
	v.builder.WriteString("(")

	for i := range len(node.Args) {
		if i > 0 {
			// 检查当前参数是否为时间单位
			_, isTimeUnit := node.Args[i].(*ast.TimeUnitExpr)
			// 检查前一个参数是否为值表达式
			_, prevIsValue := node.Args[i-1].(*test_driver.ValueExpr)
			// 只有当当前参数不是时间单位或前一个参数不是值表达式时才添加逗号
			if !isTimeUnit || !prevIsValue {
				v.builder.WriteString(", ")
			}
		}

		arg := node.Args[i]

		// 检查下一个参数是否为时间单位
		nextIsTimeUnit := false
		if i+1 < len(node.Args) {
			_, nextIsTimeUnit = node.Args[i+1].(*ast.TimeUnitExpr)
		}

		// 如果是时间单位表达式，则特殊处理
		if interval, ok := arg.(*ast.TimeUnitExpr); ok {
			if i > 0 {
				// 检查前一个参数是否为值表达式
				if _, prevIsValue := node.Args[i-1].(*test_driver.ValueExpr); prevIsValue {
					// 如果前一个参数是值表达式，我们需要将其作为参数
					if valExpr, ok := node.Args[i-1].(*test_driver.ValueExpr); ok {
						v.params = append(v.params, valExpr.GetValue())
					}
				}
			}
			v.builder.WriteString("INTERVAL ")
			v.builder.WriteString("? ")
			v.builder.WriteString(interval.Unit.String())
			continue
		}

		// 如果当前参数是值表达式，且下一个参数是时间单位，则跳过当前参数
		if _, isValue := arg.(*test_driver.ValueExpr); isValue && nextIsTimeUnit {
			continue
		}

		// 处理其他类型的参数
		arg.Accept(v)
	}

	v.builder.WriteString(")")
}

// handleUnaryOperationExpr 处理一元操作表达式
func (v *ExtractVisitor) handleUnaryOperationExpr(node *ast.UnaryOperationExpr) {
	v.builder.WriteString(node.Op.String())
	v.builder.WriteString(" ")
	node.V.Accept(v)
}

// handleIsNullExpr 处理 IS NULL 和 IS NOT NULL 表达式
func (v *ExtractVisitor) handleIsNullExpr(node *ast.IsNullExpr) {
	node.Expr.Accept(v)
	if node.Not {
		v.builder.WriteString(" IS NOT NULL")
	} else {
		v.builder.WriteString(" IS NULL")
	}
}

// handleExistsSubqueryExpr 处理 EXISTS 和 NOT EXISTS 表达式
func (v *ExtractVisitor) handleExistsSubqueryExpr(node *ast.ExistsSubqueryExpr) {
	if node.Not {
		v.builder.WriteString("NOT ")
	}
	v.builder.WriteString("EXISTS (")

	node.Sel.Accept(v)
	v.builder.WriteString(")")
}

// handleDefaultExpr 处理 DEFAULT 表达式
func (v *ExtractVisitor) handleDefaultExpr(node *ast.DefaultExpr) {
	v.builder.WriteString("DEFAULT")
	if node.Name != nil {
		v.builder.WriteString(" ")
		v.builder.WriteString(node.Name.String())
	}
}

// handleTimeUnitExpr 处理时间单位表达式
func (v *ExtractVisitor) handleTimeUnitExpr(node *ast.TimeUnitExpr) {
	_ = node
	// 不要在这里写入任何内容，因为参数占位符和 INTERVAL 关键字
	// 会在父节点（如 FuncCallExpr）中处理
}

// handleCompareSubqueryExpr 处理带有比较运算符的子查询表达式
// 例如: age > ALL(SELECT age FROM users)
func (v *ExtractVisitor) handleCompareSubqueryExpr(node *ast.CompareSubqueryExpr) {
	node.L.Accept(v)

	v.builder.WriteByte(' ')
	v.builder.WriteString(node.Op.String())

	// 添加 ALL/ANY 关键字
	if node.All {
		v.builder.WriteString(" ALL")
	} else {
		v.builder.WriteString(" ANY")
	}

	// 处理子查询
	v.builder.WriteByte('(')
	node.R.Accept(v)
	v.builder.WriteByte(')')
}

// handleShowStmt 处理 SHOW 语句
func (v *ExtractVisitor) handleShowStmt(node *ast.ShowStmt) {
	if v.opType == models.SQLOperationUnknown {
		v.opType = models.SQLOperationShow
	}

	v.builder.WriteString("SHOW ")

	// 处理不同类型的 SHOW 语句
	switch node.Tp {
	case ast.ShowCreateTable:
		v.handleShowCreateTable(node)
	case ast.ShowCreateDatabase:
		v.handleShowCreateDatabase(node)
	case ast.ShowDatabases:
		v.handleShowDatabases(node)
	case ast.ShowTables:
		v.handleShowTables(node)
	case ast.ShowColumns:
		v.handleShowColumns(node)
	case ast.ShowIndex:
		v.handleShowIndex(node)
	case ast.ShowStatus:
		v.handleShowStatus(node)
	case ast.ShowVariables:
		v.handleShowVariables(node)
	case ast.ShowProcessList:
		v.handleShowProcessList(node)
	case ast.ShowTableStatus:
		v.handleShowTableStatus(node)
	case ast.ShowWarnings, ast.ShowErrors:
		v.handleShowWarningsOrErrors(node)
	default:
		// 其他 SHOW 语句类型的处理可以在这里添加
		v.logError(fmt.Sprintf("Unhandled ShowStmt type: %v", node.Tp))
	}
}

// handleShowCreateTable 处理 SHOW CREATE TABLE 语句
func (v *ExtractVisitor) handleShowCreateTable(node *ast.ShowStmt) {
	v.builder.WriteString("CREATE TABLE ")
	if node.Table != nil {
		v.appendTableName(node.Table)
	}
}

// handleShowCreateDatabase 处理 SHOW CREATE DATABASE 语句
func (v *ExtractVisitor) handleShowCreateDatabase(node *ast.ShowStmt) {
	v.builder.WriteString("CREATE DATABASE ")
	if node.DBName != "" {
		v.builder.WriteString(node.DBName)
	}
	if node.IfNotExists {
		v.builder.WriteString(" IF NOT EXISTS")
	}
}

// handleShowDatabases 处理 SHOW DATABASES 语句
func (v *ExtractVisitor) handleShowDatabases(node *ast.ShowStmt) {
	v.builder.WriteString("DATABASES")
	v.appendPatternAndWhere(node)
}

// handleShowTables 处理 SHOW TABLES 语句
func (v *ExtractVisitor) handleShowTables(node *ast.ShowStmt) {
	v.builder.WriteString("TABLES")
	if node.DBName != "" {
		v.builder.WriteString(" FROM ")
		v.builder.WriteString(node.DBName)
	}
	v.appendPatternAndWhere(node)
}

// handleShowColumns 处理 SHOW COLUMNS 语句
func (v *ExtractVisitor) handleShowColumns(node *ast.ShowStmt) {
	v.builder.WriteString("COLUMNS FROM ")
	if node.Table != nil {
		v.appendTableName(node.Table)
	}
	v.appendPatternAndWhere(node)
}

// handleShowIndex 处理 SHOW INDEX 语句
func (v *ExtractVisitor) handleShowIndex(node *ast.ShowStmt) {
	v.builder.WriteString("INDEX FROM ")
	if node.Table != nil {
		v.appendTableName(node.Table)
	}
}

// handleShowStatus 处理 SHOW STATUS 语句
func (v *ExtractVisitor) handleShowStatus(node *ast.ShowStmt) {
	v.builder.WriteString("STATUS")
	v.appendPatternAndWhere(node)
}

// handleShowVariables 处理 SHOW VARIABLES 语句
func (v *ExtractVisitor) handleShowVariables(node *ast.ShowStmt) {
	v.builder.WriteString("VARIABLES")
	v.appendPatternAndWhere(node)
}

// handleShowProcessList 处理 SHOW PROCESSLIST 语句
func (v *ExtractVisitor) handleShowProcessList(node *ast.ShowStmt) {
	if node.Full {
		v.builder.WriteString("FULL ")
	}
	v.builder.WriteString("PROCESSLIST")
}

// handleShowTableStatus 处理 SHOW TABLE STATUS 语句
func (v *ExtractVisitor) handleShowTableStatus(node *ast.ShowStmt) {
	v.builder.WriteString("TABLE STATUS")
	if node.DBName != "" {
		v.builder.WriteString(" FROM ")
		v.builder.WriteString(node.DBName)
	}
	v.appendPatternAndWhere(node)
}

// handleShowWarningsOrErrors 处理 SHOW WARNINGS 或 SHOW ERRORS 语句
func (v *ExtractVisitor) handleShowWarningsOrErrors(node *ast.ShowStmt) {
	if node.Tp == ast.ShowWarnings {
		v.builder.WriteString("WARNINGS")
	} else {
		v.builder.WriteString("ERRORS")
	}
	if node.Limit != nil {
		node.Limit.Accept(v)
	}
}

// appendTableName 添加表名到 SQL 字符串
func (v *ExtractVisitor) appendTableName(table *ast.TableName) {
	if table.Schema.O != "" {
		v.builder.WriteString(table.Schema.O)
		v.builder.WriteString(".")
	}
	v.builder.WriteString(table.Name.O)
}

// appendPatternAndWhere 添加 LIKE 和 WHERE 子句到 SQL 字符串
func (v *ExtractVisitor) appendPatternAndWhere(node *ast.ShowStmt) {
	if node.Pattern != nil {
		v.builder.WriteString(" LIKE ")
		if valExpr, ok := node.Pattern.Pattern.(*test_driver.ValueExpr); ok {
			v.builder.WriteString("?")
			v.params = append(v.params, valExpr.GetValue())
		} else {
			node.Pattern.Pattern.Accept(v)
		}
	}
	if node.Where != nil {
		v.builder.WriteString(" WHERE ")
		node.Where.Accept(v)
	}
}

// FIXME logError logs unhandled node type errors during SQL templatization
func (v *ExtractVisitor) logError(details string) {
	msg := "[SQL Templatize Error] unhandled node type: " + details
	fmt.Println(msg)
}
