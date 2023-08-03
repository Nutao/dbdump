/*
   @Time : 2021/8/25 上午11:22
   @Author : nutao
   @File : pg_test
   @Description: NULL
*/

// Package main ...
package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

func TestQueryPG(t *testing.T) {
	connStr := "user=postgres dbname=postgres password=123456 host=xx.xx.xx.xxc port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	nameList := []interface{}{"test"}
	fmt.Println(ValuesInCondition(nameList))
	querySQL := "xxxx"

	rows, err := db.Query(querySQL)
	defer rows.Close()
	if err != nil {
		t.Fatal(err)
	}

	result := make(map[string][]*Table)
	for rows.Next() {
		t := &Table{}
		if err := rows.Scan(&t.TableSchema, &t.TableName, &t.TableType, &t.TableComment); err != nil {
			log.Fatal(err)
		}
		t.TableComment = strings.TrimSpace(t.TableComment)
		result[t.TableSchema] = append(result[t.TableSchema], t)
	}

	t.Log(result)
}

func ValuesInCondition(values []interface{}) (string, []interface{}) {
	if len(values) == 0 {
		return "0", values
	}
	return fmt.Sprintf("(?%s)", strings.Repeat(",?", len(values)-1)), values
}
