package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/urfave/cli/v2"
)

// Constraint holds constraints of table.
type Constraint struct {
	ConstraintName string
	ConstraintType string
	TableSchema    string
	TableName      string
	Enforced       string
	Columns        []string
}

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
func loadConstraints(db *sql.DB) (map[string]map[string]map[string]*Constraint, error) {
	builder := squirrel.Select("CONSTRAINT_NAME, TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME").
		From("KEY_COLUMN_USAGE")
	if gConfig.Database != "" {
		builder = builder.Where(squirrel.Eq{"TABLE_SCHEMA": gConfig.Database})
	}
	if len(gConfig.Tables) != 0 {
		builder = builder.Where(squirrel.Eq{"TABLE_NAME": gConfig.Tables})
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

	// 约束类型存储在TABLE_CONSTRAINTS表中
	defBuilder := squirrel.Select("TABLE_SCHEMA, TABLE_NAME, CONSTRAINT_NAME, CONSTRAINT_TYPE").
		From("TABLE_CONSTRAINTS")
	if gConfig.Database != "" {
		defBuilder = defBuilder.Where(squirrel.Eq{"TABLE_SCHEMA": gConfig.Database})
	}
	if len(gConfig.Tables) != 0 {
		defBuilder = defBuilder.Where(squirrel.Eq{"TABLE_NAME": gConfig.Tables})
	}

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

// Column holds the definition of a table column.
type Column struct {
	TableCatalog      string
	TableSchema       string
	TableName         string
	ColumnName        string
	OrdinalPosition   uint32
	ColumnDefaultNull bool
	ColumnDefault     string
	IsNullable        string
	DataType          string
	ColumnType        string
	ColumnKey         string
	Extra             string
	ColumnComment     string
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
func loadColumns(db *sql.DB) (map[string]map[string][]*Column, error) {
	builder := squirrel.Select("TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, " +
		"COLUMN_DEFAULT, IS_NULLABLE, DATA_TYPE, COLUMN_TYPE, COLUMN_KEY, EXTRA, COLUMN_COMMENT").From("COLUMNS")
	if gConfig.Database != "" {
		builder = builder.Where(squirrel.Eq{"TABLE_SCHEMA": gConfig.Database})
	}
	if len(gConfig.Tables) != 0 {
		builder = builder.Where(squirrel.Eq{"TABLE_NAME": gConfig.Tables})
	}
	builder = builder.OrderBy("ORDINAL_POSITION")

	rows, err := builder.RunWith(db).Query()
	if err != nil {
		return nil, fmt.Errorf("query columns info failed, %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string][]*Column)
	for rows.Next() {
		var columnDefault sql.NullString
		c := &Column{}
		if err := rows.Scan(&c.TableCatalog, &c.TableSchema, &c.TableName, &c.ColumnName, &c.OrdinalPosition,
			&columnDefault, &c.IsNullable, &c.DataType, &c.ColumnType, &c.ColumnKey, &c.Extra,
			&c.ColumnComment); err != nil {
			return nil, fmt.Errorf("scan columns failed, %w", err)
		}
		c.ColumnDefaultNull = !columnDefault.Valid
		c.ColumnDefault = columnDefault.String
		c.ColumnComment = strings.TrimSpace(c.ColumnComment)

		columnsInDB := result[c.TableSchema]
		if columnsInDB == nil {
			columnsInDB = make(map[string][]*Column)
		}
		columnsInDB[c.TableName] = append(columnsInDB[c.TableName], c)

		result[c.TableSchema] = columnsInDB
	}
	return result, nil
}

// Table holds the definition of a database table.
type Table struct {
	TableCatalog string
	TableSchema  string
	TableName    string
	TableType    string
	TableComment string

	Columns     []*Column
	Constraints []*Constraint
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
func loadTables(db *sql.DB) (map[string][]*Table, error) {
	builder := squirrel.Select("TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE, TABLE_COMMENT").
		From("TABLES")
	if gConfig.Database != "" {
		builder = builder.Where(squirrel.Eq{"TABLE_SCHEMA": gConfig.Database})
	}
	if len(gConfig.Tables) != 0 {
		builder = builder.Where(squirrel.Eq{"TABLE_NAME": gConfig.Tables})
	}

	rows, err := builder.RunWith(db).Query()
	if err != nil {
		return nil, fmt.Errorf("query tables info failed, %w", err)
	}
	defer rows.Close()

	result := make(map[string][]*Table)
	for rows.Next() {
		t := &Table{}
		if err := rows.Scan(&t.TableSchema, &t.TableName, &t.TableType, &t.TableComment); err != nil {
			return nil, fmt.Errorf("scan tables failed, %w", err)
		}
		t.TableComment = strings.TrimSpace(t.TableComment)
		result[t.TableSchema] = append(result[t.TableSchema], t)
	}
	return result, nil
}

func dump(ctx *cli.Context) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%v)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		gConfig.User, gConfig.Password, gConfig.Host, gConfig.Port, "information_schema")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("connect to database failed, %w", err)
	}
	defer db.Close()

	allTables, err := loadTables(db)
	if err != nil {
		return err
	}

	allColumns, err := loadColumns(db)
	if err != nil {
		return err
	}
	for dbName, tablesInDB := range allTables {
		for _, table := range tablesInDB {
			columnsInDB := allColumns[dbName]
			if columnsInDB != nil {
				table.Columns = columnsInDB[table.TableName]
			}
		}
	}

	allConstraints, err := loadConstraints(db)
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
