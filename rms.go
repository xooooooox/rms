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
	err := CreateModelWrite()
	if err != nil {
		log.Fatalln(err)
	}
}

// CreateModelWrite information schema write
func CreateModelWrite() error {
	tables, err := sea.InformationSchemaAllTables(DbName)
	if err != nil {
		return err
	}
	//return nil
	content := "package " + PackageName + "\n\n"
	content += fmt.Sprintf("// datetime %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
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
		content += fmt.Sprintf("// %s %s %s\n", sea.UnderlineToPascal(tableName), tableName, vt.TableComment)
		content += fmt.Sprintf("type %s struct{\n", sea.UnderlineToPascal(tableName))
		for _, vc := range columns {
			columnName := vc.ColumnName
			golangType := CreateModelColumnDataTypeToGoType(vc.DataType)
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
			content += fmt.Sprintf("\t%s %s `json:\"%s\"", sea.UnderlineToPascal(columnName), golangType, vc.ColumnName)
			if Xorm == "Y" {
				content += fmt.Sprintf(" xorm:\"%s\"", CreateTagXORM(&vc))
			}
			content += fmt.Sprintf("` // %s\n", vc.ColumnComment)
		}
		content += fmt.Sprintf("}\n\n")
	}
	err = CreateModelWriteIntoFile(&content)
	if err != nil {
		return err
	}
	return nil
}

// CreateTagXORM create xorm tag
func CreateTagXORM(c *sea.InformationSchemaColumns) string {
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

// CreateModelColumnDataTypeToGoType mysql data type to golang type
func CreateModelColumnDataTypeToGoType(dataType string) string {
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

// CreateModelWriteIntoFile write into file
func CreateModelWriteIntoFile(s *string) error {
	separator := string(filepath.Separator)
	absDir, _ := filepath.Abs("./")
	filename := absDir + separator + DbName + ".go"
	_, err := os.Stat(filename)
	// file exist
	if err == nil {
		// delete this file
		err = os.Remove(filename)
		if err != nil {
			return err
		}
	}
	// create file
	file, err := os.Create(filename)
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	if err != nil {
		return err
	}
	n, err := file.WriteString(*s)
	if err != nil {
		return err
	}
	if n < 1 {
		return errors.New("write file failed")
	}
	_ = CreateModelFmtFile(filename)
	return nil
}

// CreateModelFmtFile fmt file
func CreateModelFmtFile(file string) error {
	cmd := exec.Command("go", "fmt", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
