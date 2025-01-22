package templatize

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/test_driver"
)

const paramsMaxCount = 64

type SQLTemplatizer struct {
	parser *parser.Parser

	pool sync.Pool
}

func NewSQLTemplatizer() *SQLTemplatizer {
	return &SQLTemplatizer{
		parser: parser.New(),
		pool: sync.Pool{
			New: func() any {
				return &TemplateVisitor{
					builder: &strings.Builder{},
					params:  make([]any, 0, paramsMaxCount),
				}
			},
		},
	}
}

// TemplatizeSQL returns the templatized SQL and the parameters.
// It supports multiple SQL statements separated by semicolons.
func (p *SQLTemplatizer) TemplatizeSQL(sql string) (string, []any, error) {
	if sql == "" {
		return "", nil, fmt.Errorf("empty SQL statement")
	}

	stmts, _, err := p.parser.Parse(sql, "", "")
	if err != nil {
		return "", nil, err
	}

	if len(stmts) == 0 {
		return "", nil, fmt.Errorf("no valid SQL statements found")
	}

	// Handle multiple statements
	var result strings.Builder
	var allParams []any

	for idx := range stmts {
		if idx > 0 {
			result.WriteString("; ")
		}

		templatedSQL, params, err := p.templatizeOneStmt(stmts[idx])
		if err != nil {
			return "", nil, fmt.Errorf("error processing statement %d: %w", idx+1, err)
		}

		result.WriteString(templatedSQL)
		allParams = append(allParams, params...)
	}

	return result.String(), allParams, nil
}

// templatizeOneStmt handles a single SQL statement
func (p *SQLTemplatizer) templatizeOneStmt(stmt ast.StmtNode) (string, []any, error) {
	v := p.pool.Get().(*TemplateVisitor)
	defer func() {
		v.builder.Reset()
		v.params = v.params[:0]
		v.inAggrFunc = false

		p.pool.Put(v)
	}()

	stmt.Accept(v)
	return v.builder.String(), v.params, nil
}

// TemplateVisitor 实现 ast.Visitor 接口
type TemplateVisitor struct {
	builder    *strings.Builder
	params     []any
	inAggrFunc bool // 是否在聚合函数中
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
func (v *TemplateVisitor) Enter(n ast.Node) (ast.Node, bool) { //nolint:funlen,gocyclo
	if n == nil {
		return n, false
	}

	switch node := n.(type) {
	// 1. 基础表达式层 - 最常用的表达式处理
	case *ast.ColumnNameExpr:
		v.handleColumnNameExpr(node)
		return nil, true
	case *test_driver.ValueExpr:
		v.handleValueExpr(node)
		return nil, true
	case *ast.BinaryOperationExpr:
		v.handleBinaryOperationExpr(node)
		return nil, true
	case *ast.TableName:
		v.handleTableName(node)
		return nil, true

	// 2. SQL 语句层
	case *ast.SelectStmt:
		return v.handleSelectStmt(node)
	case *ast.InsertStmt:
		return v.handleInsertStmt(node)
	case *ast.UpdateStmt:
		return v.handleUpdateStmt(node)
	case *ast.DeleteStmt:
		return v.handleDeleteStmt(node)
	case *ast.ExplainStmt:
		return v.handleExplainStmt(node)

	// 3. 表结构层 - 表引用和连接
	case *ast.TableSource:
		v.handleTableSource(node)
		return nil, true
	case *ast.Join:
		v.handleJoin(node)
		return nil, true
	case *ast.OnCondition:
		v.handleOnCondition(node)
		return nil, true

	// 4. 条件表达式层 - WHERE/HAVING 子句中的条件
	case *ast.PatternInExpr:
		v.handlePatternInExpr(node)
		return nil, true
	case *ast.PatternLikeOrIlikeExpr:
		v.handlePatternLikeOrIlikeExpr(node)
		return nil, true
	case *ast.BetweenExpr:
		v.handleBetweenExpr(node)
		return nil, true
	case *ast.ParenthesesExpr:
		v.handleParenthesesExpr(node)
		return nil, true
	case *ast.CaseExpr:
		v.handleCaseExpr(node)
		return nil, true

	// 5. 函数和聚合层
	case *ast.FuncCallExpr:
		v.handleFuncCallExpr(node)
		return nil, true
	case *ast.AggregateFuncExpr:
		old := v.inAggrFunc
		v.inAggrFunc = true
		defer func() { v.inAggrFunc = old }()
		v.handleAggregateFuncExpr(node)
		return nil, true
	case *ast.UnaryOperationExpr:
		v.handleUnaryOperationExpr(node)
		return nil, true
	case *ast.TimeUnitExpr:
		v.handleTimeUnitExpr(node)
		return nil, true

	// 6. 修饰语层 - ORDER BY, LIMIT 等
	case *ast.ByItem:
		v.handleByItem(node)
		return nil, true
	case *ast.Limit:
		v.handleLimit(node)
		return nil, true
	case *ast.Assignment:
		v.handleAssignment(node)
		return nil, true
	case *ast.ValuesExpr:
		v.handleValuesExpr(node)
		return nil, true

	// 7. 子查询层 - 最复杂的查询结构
	case *ast.SubqueryExpr:
		v.handleSubqueryExpr(node)
		return nil, true
	case *ast.IsNullExpr:
		v.handleIsNullExpr(node)
		return nil, true
	case *ast.ExistsSubqueryExpr:
		v.handleExistsSubqueryExpr(node)
		return nil, true

	// 8. 处理 DEFAULT 表达式
	case *ast.DefaultExpr:
		v.handleDefaultExpr(node)
		return nil, true

	default:
		v.logError("unhandled node type: ", fmt.Sprintf("%T", node))
		return n, true
	}
}

// Leave 实现 ast.Visitor 接口.
// Return: n, true - 不继续遍历
func (v *TemplateVisitor) Leave(n ast.Node) (ast.Node, bool) {
	return n, true
}

// SELECT 子句: SELECT 列表、FROM 子句、WHERE 子句、GROUP BY 子句、HAVING 子句、ORDER BY 子句、LIMIT 子句
func (v *TemplateVisitor) handleSelectStmt(node *ast.SelectStmt) (ast.Node, bool) {
	if node == nil {
		return node, false
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

			if node.Fields.Fields[idx].WildCard != nil {
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
			v.handleBinaryOperationExpr(expr)
		case *ast.AggregateFuncExpr:
			v.handleAggregateFuncExpr(expr)
		default:
			fmt.Printf("HAVING expr type: %T\n", expr)
			node.Having.Expr.Accept(v)
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
		v.handleLimit(node.Limit)
	}

	return nil, true
}

// INSERT 语句
func (v *TemplateVisitor) handleInsertStmt(node *ast.InsertStmt) (ast.Node, bool) {
	if node == nil {
		return node, false
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

		for idx, item := range node.OnDuplicate {
			if idx > 0 {
				v.builder.WriteString(", ")
			}

			// item.Accept(v)
			v.handleAssignment(item)
		}
	}

	return nil, true
}

// UPDATE
func (v *TemplateVisitor) handleUpdateStmt(node *ast.UpdateStmt) (ast.Node, bool) {
	if node == nil {
		return node, false
	}

	v.builder.WriteString("UPDATE ")

	if node.TableRefs != nil && node.TableRefs.TableRefs != nil {
		node.TableRefs.TableRefs.Accept(v) // call handleTableSource()
	}

	// SET
	v.builder.WriteString(" SET ")
	for idx, assignment := range node.List {
		if idx > 0 {
			v.builder.WriteString(", ")
		}

		v.handleAssignment(assignment)
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
		v.handleLimit(node.Limit)
	}

	return nil, true
}

// DELETE
func (v *TemplateVisitor) handleDeleteStmt(node *ast.DeleteStmt) (ast.Node, bool) {
	if node == nil {
		return node, false
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
		v.handleLimit(node.Limit)
	}

	return nil, true
}

// handleTableSource 处理表源
func (v *TemplateVisitor) handleTableSource(node *ast.TableSource) {
	if node == nil {
		return
	}

	switch src := node.Source.(type) {
	case *ast.TableName:
		v.handleTableName(src)

	case *ast.SelectStmt:
		v.builder.WriteString("(")
		src.Accept(v)
		v.builder.WriteString(")")

	case *ast.Join:
		v.handleJoin(src)

	default:
		fmt.Printf("TableSource type: %T\n", src)
		node.Source.Accept(v)
	}

	if node.AsName.O != "" {
		v.builder.WriteString(" AS ")
		v.builder.WriteString(node.AsName.O)
	}
}

func (v *TemplateVisitor) handleTableName(node *ast.TableName) {
	if node == nil {
		return
	}

	if node.Schema.O != "" {
		v.builder.WriteString(node.Schema.O)
		v.builder.WriteString(".")
	}
	v.builder.WriteString(node.Name.O)
}

func (v *TemplateVisitor) handleJoin(node *ast.Join) {
	if node == nil {
		return
	}

	if node.Left != nil {
		switch left := node.Left.(type) {
		// 若左节点是 JOIN，递归处理
		case *ast.Join:
			v.handleJoin(left)

		case *ast.TableSource:
			v.handleTableSource(left)

		default:
			node.Left.Accept(v)
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
			v.handleTableSource(right)

		default:
			fmt.Printf("Join.Right type: %T\n", right)
			node.Right.Accept(v)
		}

		// ON condition
		if node.On != nil {
			v.builder.WriteString(" ON ")
			node.On.Accept(v)
		}
	}
}

func (v *TemplateVisitor) handlePatternLikeOrIlikeExpr(node *ast.PatternLikeOrIlikeExpr) {
	if node == nil {
		return
	}

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

func (v *TemplateVisitor) handlePatternInExpr(node *ast.PatternInExpr) {
	if node == nil {
		return
	}

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

func (v *TemplateVisitor) handleBinaryOperationExpr(node *ast.BinaryOperationExpr) {
	if node == nil {
		return
	}

	node.L.Accept(v)
	v.builder.WriteString(fmt.Sprintf(" %s ", node.Op.String()))
	node.R.Accept(v)
}

func (v *TemplateVisitor) handleBetweenExpr(node *ast.BetweenExpr) {
	if node == nil {
		return
	}

	node.Expr.Accept(v)

	if node.Not {
		v.builder.WriteString("NOT ")
	}

	v.builder.WriteString(" BETWEEN ")
	node.Left.Accept(v)
	v.builder.WriteString(" AND ")
	node.Right.Accept(v)
}

func (v *TemplateVisitor) handleValueExpr(node *test_driver.ValueExpr) {
	if node == nil {
		return
	}

	if v.inAggrFunc { // 在聚合函数中，直接输出值
		switch val := node.GetValue().(type) {
		case int64, uint64:
			v.builder.WriteString(fmt.Sprintf("%d", val))

		case float64:
			v.builder.WriteString(fmt.Sprintf("%f", val))

		case string:
			v.builder.WriteString(fmt.Sprintf("'%s'", val))

		case *test_driver.MyDecimal:
			v.builder.WriteString(val.String())

		default:
			fmt.Printf("ValueExpr type: %T\n", node.GetValue())
			v.builder.WriteString(fmt.Sprintf("%v", val))
		}
	} else {
		// param -> ?
		v.builder.WriteString("?")
		v.params = append(v.params, node.GetValue())
	}
}

func (v *TemplateVisitor) handleColumnNameExpr(node *ast.ColumnNameExpr) {
	if node == nil {
		return
	}

	if node.Name.Schema.O != "" {
		v.builder.WriteString(node.Name.Schema.O)
		v.builder.WriteString(".")
	}

	if node.Name.Table.O != "" {
		v.builder.WriteString(node.Name.Table.O)
		v.builder.WriteString(".")
	}

	v.builder.WriteString(node.Name.Name.O)
}

func (v *TemplateVisitor) handleByItem(node *ast.ByItem) {
	if node == nil {
		return
	}

	node.Expr.Accept(v)

	// 处理排序方向
	if node.Desc {
		v.builder.WriteString(" DESC")
	}

	// FIXME 处理 NULL 排序
}

func (v *TemplateVisitor) handleValuesExpr(node *ast.ValuesExpr) {
	if node == nil {
		return
	}

	v.builder.WriteString("VALUES(")
	node.Column.Accept(v)
	// node.Accept(v)
	v.builder.WriteString(")")
}

func (v *TemplateVisitor) handleLimit(node *ast.Limit) {
	if node == nil {
		return
	}

	v.builder.WriteString(" LIMIT ")

	if node.Offset != nil {
		node.Offset.Accept(v)
		v.builder.WriteString(", ")
	}

	node.Count.Accept(v)
}

func (v *TemplateVisitor) handleSubqueryExpr(node *ast.SubqueryExpr) {
	if node == nil {
		return
	}

	if v.builder.String()[v.builder.Len()-1] == '(' {
		node.Query.Accept(v)
		return
	}

	v.builder.WriteString("(")
	node.Query.Accept(v)
	v.builder.WriteString(")")
}

func (v *TemplateVisitor) handleOnCondition(node *ast.OnCondition) {
	if node == nil {
		return
	}

	node.Expr.Accept(v)
}

// handleAssignment 处理赋值表达式
func (v *TemplateVisitor) handleAssignment(node *ast.Assignment) {
	if node == nil {
		return
	}

	v.handleColumnNameExpr(&ast.ColumnNameExpr{Name: node.Column})
	v.builder.WriteString(" eq ")
	node.Expr.Accept(v)
}

// handleExprNode 处理表达式节点
func (v *TemplateVisitor) handleAggregateFuncExpr(node *ast.AggregateFuncExpr) {
	if node == nil {
		return
	}

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
func (v *TemplateVisitor) handleCaseExpr(node *ast.CaseExpr) {
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
func (v *TemplateVisitor) handleParenthesesExpr(node *ast.ParenthesesExpr) {
	v.builder.WriteString("(")
	node.Expr.Accept(v)
	v.builder.WriteString(")")
}

// handleFuncCallExpr 处理函数调用表达式
func (v *TemplateVisitor) handleFuncCallExpr(node *ast.FuncCallExpr) {
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
			v.builder.WriteString("?")
			v.builder.WriteString(" ")
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
func (v *TemplateVisitor) handleUnaryOperationExpr(node *ast.UnaryOperationExpr) {
	v.builder.WriteString(node.Op.String())
	v.builder.WriteString(" ")
	node.V.Accept(v)
}

// handleIsNullExpr 处理 IS NULL 和 IS NOT NULL 表达式
func (v *TemplateVisitor) handleIsNullExpr(node *ast.IsNullExpr) {
	node.Expr.Accept(v)
	if node.Not {
		v.builder.WriteString(" IS NOT NULL")
	} else {
		v.builder.WriteString(" IS NULL")
	}
}

// handleExistsSubqueryExpr 处理 EXISTS 和 NOT EXISTS 表达式
func (v *TemplateVisitor) handleExistsSubqueryExpr(node *ast.ExistsSubqueryExpr) {
	if node.Not {
		v.builder.WriteString("NOT ")
	}
	v.builder.WriteString("EXISTS (")

	node.Sel.Accept(v)
	v.builder.WriteString(")")
}

// handleDefaultExpr 处理 DEFAULT 表达式
func (v *TemplateVisitor) handleDefaultExpr(node *ast.DefaultExpr) {
	v.builder.WriteString("DEFAULT")
	if node.Name != nil {
		v.builder.WriteString(" ")
		v.builder.WriteString(node.Name.String())
	}
}

// handleTimeUnitExpr 处理时间单位表达式
func (v *TemplateVisitor) handleTimeUnitExpr(node *ast.TimeUnitExpr) {
	// 不要在这里写入任何内容，因为参数占位符和 INTERVAL 关键字
	// 会在父节点（如 FuncCallExpr）中处理
}

// handleExplainStmt 处理 EXPLAIN 语句
func (v *TemplateVisitor) handleExplainStmt(node *ast.ExplainStmt) (ast.Node, bool) {
	if node == nil {
		return node, false
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

	return nil, true
}

// FIXME logError 统一的错误日志处理
func (v *TemplateVisitor) logError(errType string, details string) {
	msg := fmt.Sprintf("[SQL Templatize Error] %s: %s", errType, details)
	fmt.Println(msg)
}
