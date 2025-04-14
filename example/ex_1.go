package main

import (
	"fmt"
	"log"

	sqlextractor "github.com/kydance/sql-extractor"
)

func main() {
	// 创建提取器
	extractor := sqlextractor.NewExtractor(
		"SELECT * FROM users WHERE age > 18 AND name LIKE 'John%';SELECT a FROM t1 WHERE a = 1 AND b = 2;")

	// 提取 SQL 信息
	err := extractor.Extract()
	if err != nil {
		log.Fatal(err)
	}

	// 获取处理结果
	fmt.Printf("模板化 SQL: %v\n", extractor.TemplatizedSQL()) // 返回 []string
	fmt.Printf("参数: %v\n", extractor.Params())              // 返回 [][]any

	fmt.Printf("操作类型: %v\n", extractor.OpType()) // 返回 []models.SQLOpType
	fmt.Printf("%T\n", extractor.OpType())

	tis := extractor.TableInfos()
	fmt.Printf("表信息: %v\n", tis) // 返回 [][]*models.TableInfo

	for idx, ti := range tis {
		fmt.Printf("第 %d 个SQL表信息\n", idx)
		for _, tb := range ti {
			fmt.Println("Schema: ", tb.Schema())
			fmt.Println("TableName: ", tb.TableName())
		}
	}
}
