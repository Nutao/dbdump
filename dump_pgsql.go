package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/urfave/cli/v2"
)

// loadConstraints loads constraints from database
//
// loadTables loads table info from database
//
// The format of result is：
//
//           /-- database1
//          /                 /- table1  /- constraint1
// result --  -- database2 -- -- table2 --  constraint2 -- [column1, column2]
//          \                 \- table3  \- constraint3
//           \-- database3
func loadPGSQLConstraints(db *sql.DB) (map[string]map[string]map[string]*Constraint, error) {
	builder := squirrel.Select("CONSTRAINT_NAME, TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME").
		From("information_schema.KEY_COLUMN_USAGE")
	if gConfig.Database != "" {
		builder = builder.Where(squirrel.Eq{"TABLE_CATALOG": gConfig.Database}).PlaceholderFormat(squirrel.Dollar)
	}
	if len(gConfig.Tables) != 0 {
		builder = builder.Where(squirrel.Eq{"TABLE_NAME": gConfig.Tables}).PlaceholderFormat(squirrel.Dollar)
	}
	builder = builder.OrderBy("ORDINAL_POSITION")

	rows, err := builder.RunWith(db).Query()
	if err != nil {
		return nil, fmt.Errorf("query constraints info failed, %w", err)
	}
	defer rows.Close()
	result := make(map[string]map[string]map[string]*Constraint)
	for rows.Next() {
		var column string
		c := &Constraint{}
		if err := rows.Scan(&c.ConstraintName, &c.TableSchema, &c.TableName, &column); err != nil {
			return nil, fmt.Errorf("scan constraints failed, %w", err)
		}
		constraintsInDB := result[c.TableSchema]
		if constraintsInDB == nil {
			constraintsInDB = make(map[string]map[string]*Constraint)
			result[c.TableSchema] = constraintsInDB
		}
		constraintsInTable := constraintsInDB[c.TableName]
		if constraintsInTable == nil {
			constraintsInTable = make(map[string]*Constraint)
			constraintsInDB[c.TableName] = constraintsInTable
		}
		constraint := constraintsInTable[c.ConstraintName]
		if constraint == nil {
			constraint = c
			constraintsInTable[c.ConstraintName] = constraint
		}
		constraint.Columns = append(constraint.Columns, column)
	}

	// 取出上一步中的schema
	var schemaList = make([]string, 0)
	for key, _ := range result {
		schemaList = append(schemaList, key)
	}

	// 约束类型存储在TABLE_CONSTRAINTS表中
	defBuilder := squirrel.Select("TABLE_SCHEMA, TABLE_NAME, CONSTRAINT_NAME, CONSTRAINT_TYPE").
		From("information_schema.TABLE_CONSTRAINTS")
	if gConfig.Database != "" {
		defBuilder = defBuilder.Where(squirrel.Eq{"TABLE_CATALOG": gConfig.Database}).PlaceholderFormat(squirrel.Dollar)
	}

	if len(gConfig.Tables) != 0 {
		defBuilder = defBuilder.Where(squirrel.Eq{"TABLE_NAME": gConfig.Tables}).PlaceholderFormat(squirrel.Dollar)
	}

	defBuilder.Where(squirrel.Eq{"TABLE_SCHEMA": schemaList}).PlaceholderFormat(squirrel.Dollar)

	// fmt.Println(defBuilder.ToSql())
	defRows, err := defBuilder.RunWith(db).Query()
	if err != nil {
		return nil, fmt.Errorf("query constraints define failed, %w", err)
	}
	defer defRows.Close()
	for defRows.Next() {
		var tableSchema, tableName, cName, cType string
		if err := defRows.Scan(&tableSchema, &tableName, &cName, &cType); err != nil {
			return nil, fmt.Errorf("scan constraint defines failed, %w", err)
		}
		constraintsInDB := result[tableSchema]
		if constraintsInDB == nil {
			continue
		}
		constraintsInTable := constraintsInDB[tableName]
		if constraintsInTable == nil {
			continue
		}
		constraint := constraintsInTable[cName]
		if constraint != nil {
			constraint.ConstraintType = cType
		}
	}

	return result, nil
}

// loadColumns loads columns from database
//
// loadTables loads table info from database
//
// The format of result is：
//
//           /-- database1
//          /                 /- table1
// result --  -- database2 -- -- table2 -- [column1, column2]
//          \                 \- table3
//           \-- database3
// 弃用
// func loadPGSQLColumns(db *sql.DB) (map[string]map[string][]*Column, error) {
// 	builder := squirrel.Select("a.attnum as oid, c.relname as tableName, COALESCE(CAST(obj_description(relfilenode, 'pg_class') as VARCHAR ), 'NULL' )as tableDesc, a.attname as columnName, concat_ws('', t.typname, SUBSTRING(format_type(a.atttypid, a.atttypmod) FROM '\\(.*\\)')) as columnType, d.description as columnComment").
// 		From("pg_class as c, pg_attribute as a, pg_type as t, pg_description as d").
// 		Where("a.attnum > 0 AND a.attrelid = c.oid AND a.atttypid = t.oid AND d.objoid = a.attrelid AND d.objsubid = a.attnum")
//
// 	if len(gConfig.Tables) != 0 {
// 		builder = builder.Where(squirrel.Eq{" c.relname": gConfig.Tables}).PlaceholderFormat(squirrel.Dollar)
// 	}
//
// 	builder = builder.OrderBy("c.relname").OrderBy("a.attnum")
//
// 	rows, err := builder.RunWith(db).Query()
// 	if err != nil {
// 		return nil, fmt.Errorf("query columns info failed, %w", err)
// 	}
// 	defer rows.Close()
//
// 	result := make(map[string]map[string][]*Column)
// 	for rows.Next() {
// 		// var columnDefault sql.NullString
// 		c := &Column{}
// 		if err := rows.Scan(
// 			&c.OrdinalPosition,
// 			&c.TableName,
// 			&c.Extra,
// 			&c.ColumnName,
// 			&c.ColumnType,
// 			&c.ColumnComment,
// 		); err != nil {
// 			return nil, fmt.Errorf("scan columns failed, %w", err)
// 		}
// 		// c.ColumnDefaultNull = !columnDefault.Valid
// 		// c.ColumnDefault = columnDefault.String
// 		c.ColumnComment = strings.TrimSpace(c.ColumnComment)
// 		c.TableSchema = "public"
// 		columnsInDB := result[c.TableSchema]
// 		if columnsInDB == nil {
// 			columnsInDB = make(map[string][]*Column)
// 		}
// 		columnsInDB[c.TableName] = append(columnsInDB[c.TableName], c)
//
// 		result[c.TableSchema] = columnsInDB
// 	}
// 	return result, nil
// }

func loadPGSQLColumnsNew(db *sql.DB, table string) (map[string]map[string][]*Column, error) {
	builder := squirrel.Select("A.attname AS COLUMN_NAME,"+
		"concat_ws('', t.typname, SUBSTRING(format_type(a.atttypid, a.atttypmod) FROM '\\(.*\\)')), "+
		"(CASE WHEN (SELECT COUNT (*) FROM pg_constraint WHERE conrelid=A.attrelid AND conkey [ 1 ]=attnum AND contype='p')> 0 THEN 'Y' ELSE 'N' END) AS 主键约束,"+
		"(CASE WHEN (SELECT COUNT (*) FROM pg_constraint WHERE conrelid=A.attrelid AND conkey [ 1 ]=attnum AND contype='u')> 0 THEN 'Y' ELSE 'N' END) AS 唯一约束,"+
		"(CASE WHEN (SELECT COUNT (*) FROM pg_constraint WHERE conrelid=A.attrelid AND conkey [ 1 ]=attnum AND contype='f')> 0 THEN 'Y' ELSE 'N' END) AS 外键约束,"+
		"(CASE WHEN A.attnotnull=TRUE THEN 'N' ELSE 'Y' END) AS NULLABLE,"+
		"col_description (A.attrelid,A.attnum) AS COMMENT ").
		From("pg_attribute a, pg_type t").
		Where("a.atttypid = t.oid AND attstattarget=-1 and attrelid = (select oid from pg_class where relname = $1)", table).
		OrderBy("a.attnum ASC")

	rows, err := builder.RunWith(db).Query()
	if err != nil {
		return nil, fmt.Errorf("query columns info failed, %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string][]*Column)
	for rows.Next() {
		// var columnDefault sql.NullString
		c := &Column{}
		// 接收约束
		var pk, uk, fk string

		c.TableName = table
		if err := rows.Scan(
			&c.ColumnName,
			&c.ColumnType,
			&pk,
			&uk,
			&fk,
			&c.IsNullable,
			&c.ColumnComment,
		); err != nil {
			return nil, fmt.Errorf("scan columns failed, %w", err)
		}
		// 处理约束赋值
		if pk == "Y" {
			c.ColumnKey = "PRI KEY"
		} else if uk == "Y" {
			c.ColumnKey = "UNIQUE KEY"
		} else if fk == "Y" {
			c.ColumnKey = "FOREIGN KEY"
		}

		c.ColumnComment = strings.TrimSpace(c.ColumnComment)
		c.TableSchema = "public"
		columnsInDB := result[c.TableSchema]
		if columnsInDB == nil {
			columnsInDB = make(map[string][]*Column)
		}
		columnsInDB[c.TableName] = append(columnsInDB[c.TableName], c)

		result[c.TableSchema] = columnsInDB
	}
	return result, nil
}

// loadTables loads table info from database
//
// The format of result is：
//
//           /-- database1
//          /
// result --  -- database2 -- [table1, table2]
//          \
//           \-- database3
func loadPGSQLTables(db *sql.DB) (map[string][]*Table, error) {
	builder := squirrel.Select("TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE").
		From("information_schema.TABLES")
	if gConfig.Database != "" {
		builder = builder.Where(squirrel.Eq{"TABLE_CATALOG": gConfig.Database}).PlaceholderFormat(squirrel.Dollar)
	}
	builder = builder.Where(squirrel.Eq{"TABLE_SCHEMA": "public"}).PlaceholderFormat(squirrel.Dollar)

	if len(gConfig.Tables) != 0 {
		builder = builder.Where(squirrel.Eq{"TABLE_NAME": gConfig.Tables}).PlaceholderFormat(squirrel.Dollar)
	}
	// fmt.Println(builder.ToSql())
	rows, err := builder.RunWith(db).Query()
	if err != nil {
		return nil, fmt.Errorf("query tables info failed, %w", err)
	}
	defer rows.Close()

	result := make(map[string][]*Table)
	for rows.Next() {
		t := &Table{}
		if err := rows.Scan(&t.TableCatalog, &t.TableSchema, &t.TableName, &t.TableType); err != nil {
			return nil, fmt.Errorf("scan tables failed, %w", err)
		}
		t.TableComment = strings.TrimSpace(t.TableComment)
		result[t.TableSchema] = append(result[t.TableSchema], t)
	}
	return result, nil
}

// 导出postgres表结构。。。loadTables、loadConstrans函数一样。
func dumpPGSQL(ctx *cli.Context) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%v/%s?sslmode=disable",
		gConfig.User, gConfig.Password, gConfig.Host, gConfig.Port, gConfig.Database)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("connect to postgres database failed, %w", err)
	}
	defer db.Close()

	allTables, err := loadPGSQLTables(db)
	if err != nil {
		return err
	}

	// allColumns, err := loadPGSQLColumns(db)
	// if err != nil {
	// 	return err
	// }
	for dbName, tablesInDB := range allTables {
		for _, table := range tablesInDB {
			pgsqlColumnsNew, err := loadPGSQLColumnsNew(db, table.TableName)
			if err != nil {
				return err
			}
			columnsInDB := pgsqlColumnsNew[dbName]
			if columnsInDB != nil {
				table.Columns = columnsInDB[table.TableName]
			}
		}
	}

	allConstraints, err := loadPGSQLConstraints(db)
	if err != nil {
		return err
	}
	for dbName, tablesInDB := range allTables {
		for _, table := range tablesInDB {
			constraintsInDB := allConstraints[dbName]
			if constraintsInDB == nil {
				continue
			}
			for _, constraint := range constraintsInDB[table.TableName] {
				table.Constraints = append(table.Constraints, constraint)
			}
		}
	}

	data, err := gConfig.Formatter.Format(allTables)
	if err != nil {
		return fmt.Errorf("formate output failed, %w", err)
	}
	if gConfig.Output != "" {
		if err := ioutil.WriteFile(gConfig.Output, data, 0644); err != nil {
			return fmt.Errorf("write to output file failed, %w", err)
		}
	} else {
		fmt.Println(string(data))
	}

	return nil
}

func ValuesInCondition(values []interface{}) (string, []interface{}) {
	if len(values) == 0 {
		return "0", values
	}
	return fmt.Sprintf("(?%s)", strings.Repeat(",?", len(values)-1)), values
}
