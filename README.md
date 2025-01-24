# sql-extractor

sql-extractor 是一个高性能的 SQL 解析和转换工具，它可以将 SQL 语句转换为参数化模板，并提取相关的表信息和参数值。该工具基于 TiDB 的 SQL 解析器，支持复杂的 SQL 语句分析。

## 特性

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

## 安装

```bash
go get github.com/kydance/sql-extractor
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
    extractor := sqlextractor.NewExtractor("SELECT * FROM users WHERE age > 18 AND name LIKE 'John%'")
    
    // 提取 SQL 信息
    err := extractor.Extract()
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取处理结果
    fmt.Printf("模板化 SQL: %s\n", extractor.TemplatizeSQL())
    fmt.Printf("参数: %v\n", extractor.Params())
    fmt.Printf("表信息: %v\n", extractor.TableInfos())
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
```

## API 文档

### Extractor

主要的提取器结构体，用于处理 SQL 语句。

```go
type Extractor struct {
    // 包含已过滤或未导出的字段
}

// 创建新的提取器
func NewExtractor(sql string) *Extractor

// 提取 SQL 信息
func (e *Extractor) Extract() error

// 获取原始 SQL
func (e *Extractor) RawSQL() string

// 获取模板化后的 SQL
func (e *Extractor) TemplatizeSQL() string

// 获取提取的参数
func (e *Extractor) Params() []any

// 获取表信息
func (e *Extractor) TableInfos() []*models.TableInfo
```

### TableInfo

表信息结构体，包含 schema 和表名信息。

```go
type TableInfo struct {
    Schema    string // 数据库 schema
    TableName string // 表名
}
```

## 性能优化

- 使用 sync.Pool 复用 visitor 对象，减少内存分配
- 预分配适当大小的切片，避免频繁扩容
- 使用 strings.Builder 进行字符串拼接

## 限制说明

- 仅支持 MySQL 语法
- 不支持存储过程和函数
- 不处理注释内容

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 作者

- [@kydance](https://github.com/kydance)

## 致谢

- [TiDB Parser](https://github.com/pingcap/tidb) - SQL 解析器
