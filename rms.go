// Copyright (C) xooooooox

// Reverse Mysql/MariaDB structure

package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/xooooooox/sea"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	// DataSourceName
	DataSourceName string

	// PackageName
	PackageName string

	// DbName
	DbName string

	// xorm
	Xorm string
)

func init() {
	flag.StringVar(&DataSourceName, "s", "root:root@tcp(127.0.0.1:3306)/mysql?charset=utf8mb4", "data source name")
	flag.StringVar(&PackageName, "p", "model", "package name")
	flag.StringVar(&DbName, "d", "mysql", "database name")
	flag.StringVar(&Xorm, "x", "N", "whether to add xorm tag?(Y/N)")
	flag.Parse()
}

func init() {
	err := errors.New("error: Cannot connect to database")
	db, err := sql.Open("mysql", DataSourceName)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalln(err)
	}
	sea.SetDbInstance(db)
}

func main() {
	err := WriteStructure()
	if err != nil {
		log.Fatalln(err)
	}
}

// WriteStructure information schema write into golang structure
func WriteStructure() error {
	tables, err := sea.InformationSchemaAllTables(DbName)
	if err != nil {
		return err
	}
	//return nil
	code := "// Copyright (C) xooooooox\n\n"
	code += "package " + PackageName + "\n\n"
	code += fmt.Sprintf("// datetime %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	lengthTable := len(tables)
	if lengthTable == 0 {
		return errors.New("haven't any table")
	}
	for _, vt := range tables {
		tableName := vt.TableName
		columns, _ := sea.InformationSchemaAllColumns(vt.TableSchema, vt.TableName)
		lengthColumn := len(columns)
		if lengthColumn == 0 {
			continue
		}
		code += fmt.Sprintf("// %s %s %s\n", sea.UnderlineToPascal(tableName), tableName, vt.TableComment)
		code += fmt.Sprintf("type %s struct{\n", sea.UnderlineToPascal(tableName))
		for _, vc := range columns {
			columnName := vc.ColumnName
			golangType := ColumnDataTypeToGoType(vc.DataType)
			// first
			// golang base data type , exist 'unsigned' and 'int' keyword (integer may be unsigned)
			if strings.Index(strings.ToLower(vc.ColumnType), "unsigned") > 0 && strings.Index(strings.ToLower(vc.ColumnType), "int") > 0 {
				golangType = "u" + golangType
			}
			// twice
			// current column allow null, set type is *type or (sql.NullInt64, sql.NullFloat64, sql.NullString ...), otherwise it causes rows.Scan panic
			if strings.ToUpper(vc.IsNullable) == "YES" {
				golangType = "*" + golangType
			}
			code += fmt.Sprintf("\t%s %s `json:\"%s\"", sea.UnderlineToPascal(columnName), golangType, vc.ColumnName)
			if Xorm == "Y" {
				code += fmt.Sprintf(" xorm:\"%s\"", TagXorm(&vc))
			}
			code += fmt.Sprintf("` // %s\n", vc.ColumnComment)
		}
		code += fmt.Sprintf("}\n\n")
		err = WriteGoCode(&vt)
		if err != nil {
			return err
		}
	}
	return WriteFile(DbName+".go", &code)
}

// TagXorm create xorm tag
func TagXorm(c *sea.InformationSchemaColumns) string {
	content := ``
	if strings.ToLower(c.Extra) == `auto_increment` {
		content += `autoincr `
	}
	if strings.ToLower(c.ColumnKey) == `pri` {
		content += `pk `
	}
	if strings.ToLower(c.ColumnKey) == `uni` {
		content += `unique `
	}
	if strings.ToLower(c.ColumnKey) == `mul` {
		content += `index `
	}
	content += c.ColumnType + ` `
	if strings.ToLower(c.IsNullable) == "no" {
		content += `not null `
		if c.ColumnDefault != nil {
			columnDefault := *c.ColumnDefault
			if columnDefault == `0` {
				content += `default 0 `
			}
			if columnDefault == `''` {
				content += `default '' `
			}
		}
	} else {
		content += `default null `
	}
	// content += `comment:'` + c.ColumnComment + `' `
	content = strings.TrimRight(content, ` `)
	return content
}

// ColumnDataTypeToGoType mysql data type to golang type
func ColumnDataTypeToGoType(dataType string) string {
	switch strings.ToLower(dataType) {
	case "tinyint":
		return "int8"
	case "smallint":
		return "int16"
	case "int", "integer", "mediumint":
		return "int"
	case "float", "double", "decimal":
		return "float64"
	case "bigint":
		return "int64"
	default:
		return "string"
	}
}

// WriteGoCode quickly curd golang code
func WriteGoCode(table *sea.InformationSchemaTables) error {
	Table := sea.UnderlineToPascal(table.TableName)
	code := "// Copyright (C) xooooooox\n\n"
	code = fmt.Sprintf("%s// The following methods are only for single table operations\n\n", code)
	code = fmt.Sprintf("%spackage "+PackageName+"\n\n", code)
	code += fmt.Sprintf("// datetime %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	//code = fmt.Sprintf("%simport (\n\t\"github.com/xooooooox/sea\"\n)\n\n", code)
	//code = fmt.Sprintf("%s// Tips: insert one row return id and an error, or insert more rows return affected rows and an error\n", code)
	//code = fmt.Sprintf("%s// Insert%s insert one or more rows into `%s`\n", code, Table, table.TableName)
	//code = fmt.Sprintf("%sfunc Insert%s(insert ...interface{}) (int64, error) {\n", code, Table)
	//code = fmt.Sprintf("%s\tif len(insert) == 1 {\n", code)
	//code = fmt.Sprintf("%s\t\treturn sea.InsertOne(insert[0])\n", code)
	//code = fmt.Sprintf("%s\t}\n", code)
	//code = fmt.Sprintf("%s\treturn sea.Insert(insert...)\n", code)
	//code = fmt.Sprintf("%s}\n\n", code)

	code = fmt.Sprintf("%s// Tips: sql where condition write into the first arg of args, sql arguments are based on args[1] to the end, return affected rows and an error\n", code)
	code = fmt.Sprintf("%s// Delete%s delete one or more rows from `%s`\n", code, Table, table.TableName)
	code = fmt.Sprintf("%sfunc Delete%s(args ...interface{}) (int64, error) {\n", code, Table)
	code = fmt.Sprintf("%s\treturn sea.Delete(&%s{}, args...)\n", code, Table)
	code = fmt.Sprintf("%s}\n\n", code)

	code = fmt.Sprintf("%s// Tips: update arg is the field that needs to be updated; sql where condition write into the first arg of args, sql arguments are based on args[1] to the end, return affected rows and an error\n", code)
	code = fmt.Sprintf("%s// Update%s update one or more rows from `%s`\n", code, Table, table.TableName)
	code = fmt.Sprintf("%sfunc Update%s(update map[string]interface{}, args ...interface{}) (int64, error) {\n", code, Table)
	code = fmt.Sprintf("%s\treturn sea.Update(&%s{}, update, args...)\n", code, Table)
	code = fmt.Sprintf("%s}\n\n", code)

	code = fmt.Sprintf("%s// Tips: it is strongly recommended that the id column be the unique key, preferably the primary key\n", code)
	code = fmt.Sprintf("%s// DeleteById%s delete one row from `%s`\n", code, Table, table.TableName)
	code = fmt.Sprintf("%sfunc DeleteById%s(id int64) (int64, error) {\n", code, Table)
	code = fmt.Sprintf("%s\treturn sea.Delete(&%s{},\"`id`=?\",id)\n", code, Table)
	code = fmt.Sprintf("%s}\n\n", code)

	code = fmt.Sprintf("%s// Tips: it is strongly recommended that the id column be the unique key, preferably the primary key\n", code)
	code = fmt.Sprintf("%s// UpdateById%s update one rows from `%s`\n", code, Table, table.TableName)
	code = fmt.Sprintf("%sfunc UpdateById%s(update map[string]interface{}, id int64) (int64, error) {\n", code, Table)
	code = fmt.Sprintf("%s\treturn sea.Update(&%s{}, update, \"`id`=?\",id)\n", code, Table)
	code = fmt.Sprintf("%s}\n\n", code)

	code = fmt.Sprintf("%s// Tips: execute a query sql\n", code)
	code = fmt.Sprintf("%s// Select%s select rows from `%s`\n", code, Table, table.TableName)
	code = fmt.Sprintf("%sfunc Select%s(query string, args ...interface{}) ([]%s, error) {\n", code, Table, Table)
	code = fmt.Sprintf("%s\trows := []%s{}\n", code, Table)
	code = fmt.Sprintf("%s\terr := sea.Select(&rows, query, args...)\n", code)
	code = fmt.Sprintf("%s\treturn rows, err\n", code)
	code = fmt.Sprintf("%s}\n\n", code)

	code = fmt.Sprintf("%s// Tips: execute a query sql\n", code)
	code = fmt.Sprintf("%s// SelectOne%s select one row from `%s`\n", code, Table, table.TableName)
	code = fmt.Sprintf("%sfunc SelectOne%s(query string, args ...interface{}) (*%s, error) {\n", code, Table, Table)
	code = fmt.Sprintf("%s\trow := %s{}\n", code, Table)
	code = fmt.Sprintf("%s\terr := sea.Select(&row, query, args...)\n", code)
	code = fmt.Sprintf("%s\treturn &row, err\n", code)
	code = fmt.Sprintf("%s}\n\n", code)

	return WriteFile(DbName+"___"+table.TableName+".go", &code)
}

// WriteFile write into file
func WriteFile(file string, s *string) error {
	ds := string(filepath.Separator)
	abs, _ := filepath.Abs("./")
	file = abs + ds + file
	_, err := os.Stat(file)
	// file exist
	if err == nil {
		// delete this file
		err = os.Remove(file)
		if err != nil {
			return err
		}
	}
	// create file
	f, err := os.Create(file)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	if err != nil {
		return err
	}
	_, err = f.WriteString(*s)
	if err != nil {
		return err
	}
	return FmtFile(file)
}

// FmtFile fmt file
func FmtFile(file string) error {
	cmd := exec.Command("go", "fmt", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
