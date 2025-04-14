# sql-extractor

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue)](https://golang.org/doc/devel/release.html#go1.23)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)

sql-extractor 是一个高性能的 SQL 解析和转换工具，它可以将 SQL 语句转换为参数化模板，并提取相关的表信息和参数值。该工具基于 TiDB 的 SQL 解析器，支持复杂的 SQL 语句分析。

## 功能特性

- 支持多种 SQL 操作类型：SELECT、INSERT、UPDATE、DELETE
- SQL 语句参数化：将字面值转换为占位符
- 表信息提取：捕获查询中使用的 schema 和表名
- 参数提取：按出现顺序收集 SQL 中的字面值
- 多语句支持：可以处理以分号分隔的多个 SQL 语句
- 线程安全：使用 sync.Pool 进行并发处理
- 支持复杂 SQL 特性：
  - JOIN 操作（LEFT JOIN、RIGHT JOIN、INNER JOIN）
  - 子查询
  - 聚合函数
  - 各种 SQL 表达式（LIKE、IN、BETWEEN 等）

## 性能优化

- 使用 sync.Pool 复用 visitor 对象，减少内存分配
- 预分配适当大小的切片，避免频繁扩容
- 使用 strings.Builder 进行字符串拼接

## 系统要求

- Go 1.23 或更高版本
- 依赖包：
  - github.com/pingcap/tidb/pkg/parser
  - github.com/kydance/ziwi

## 安装

```bash
go install github.com/kydance/sql-extractor@latest
```

## 快速开始

### 基础用法

```go
package main

import (
    "fmt"
    "log"
    sqlextractor "github.com/kydance/sql-extractor"
)

func main() {
    // 创建提取器
    extractor := sqlextractor.NewExtractor(
        "SELECT * FROM users WHERE age > 18 AND name LIKE 'John%'")
    
    // 提取 SQL 信息
    err := extractor.Extract()
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取处理结果
    fmt.Printf("模板化 SQL: %v\n", extractor.TemplatizedSQL()) // 返回 []string
    fmt.Printf("参数: %v\n", extractor.Params())              // 返回 [][]any
    fmt.Printf("表信息: %v\n", extractor.TableInfos())         // 返回 [][]*models.TableInfo
    fmt.Printf("操作类型: %v\n", extractor.OpType())           // 返回 []models.SQLOpType
}
```

### 处理多条 SQL 语句

```go
sql := `
    SELECT * FROM users WHERE status = 1;
    UPDATE orders SET status = 'completed' WHERE id = 1000;
`
extractor := sqlextractor.NewExtractor(sql)
err := extractor.Extract()
if err != nil {
    log.Fatal(err)
}

// 每个切片索引对应一条 SQL 语句的结果
for i, sql := range extractor.TemplatizedSQL() {
    fmt.Printf("SQL %d:\n", i+1)
    fmt.Printf("  模板: %s\n", sql)
    fmt.Printf("  参数: %v\n", extractor.Params()[i])
    fmt.Printf("  表信息: %v\n", extractor.TableInfos()[i])
    fmt.Printf("  操作类型: %v\n", extractor.OpType()[i])
}
```

## API 文档

### Extractor

主要的提取器结构体，用于处理 SQL 语句。

```go
type Extractor struct {
    // 包含已过滤或未导出的字段
}

// NewExtractor 创建新的提取器
func NewExtractor(sql string) *Extractor

// Extract 提取 SQL 信息
func (e *Extractor) Extract() error

// RawSQL 获取原始 SQL
func (e *Extractor) RawSQL() string

// SetRawSQL 设置原始 SQL
func (e *Extractor) SetRawSQL(sql string)

// TemplatizedSQL 获取模板化后的 SQL 列表
func (e *Extractor) TemplatizedSQL() []string

// Params 获取提取的参数列表
func (e *Extractor) Params() [][]any

// TableInfos 获取表信息列表
func (e *Extractor) TableInfos() [][]*models.TableInfo

// OpType 获取 SQL 操作类型列表
func (e *Extractor) OpType() []models.SQLOpType
```

### TableInfo

表信息结构体，包含 schema 和表名信息。

```go
type TableInfo struct {
    Schema    string // 数据库 schema
    TableName string // 表名
}
```

## 贡献指南

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m 'feat: add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启一个 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 作者

- [@kydance](https://github.com/kydance)

## 致谢

- [TiDB Parser](https://github.com/pingcap/tidb) - SQL 解析器
