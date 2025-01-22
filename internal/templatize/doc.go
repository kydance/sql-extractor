// Package templatize provides functionality for SQL templatization,
// allowing SQL statements to be converted into a template format with
// extracted parameters. This is useful for preparing SQL statements
// for execution in a parameterized manner, enhancing security and
// efficiency.
//
// The package supports various SQL operations including SELECT, INSERT,
// UPDATE, DELETE, and more complex expressions like JOINs, CASE statements,
// and subqueries. It utilizes the TiDB parser for SQL parsing and AST
// traversal.
//
// Key Components:
// - SQLTemplatizer: The main struct that provides methods to templatize SQL
//   statements.
// - TemplateVisitor: Implements the ast.Visitor interface to traverse and
//   process SQL AST nodes, handling different SQL constructs.
//
// Example usage:
//   templatizer := templatize.NewSQLTemplatizer()
//   templateSQL, params, err := templatizer.TemplatizeSQL("SELECT * FROM users WHERE id = 1")
//
// This package is particularly useful for applications that require dynamic
// SQL generation and execution, ensuring that SQL statements are executed
// safely and efficiently.

package templatize
