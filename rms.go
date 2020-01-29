// Copyright (C) xooooooox

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

// CommandLineArgs Command line args
type CommandLineArgs struct {
	DatabaseSourceName string
	FilePackageName    string
	DatabaseName       string
	FmtFile            bool
	Json               bool
	Xorm               bool
	Version            bool
	FileSaveDir        string
	FileNameSuffix     string
}

var (
	args CommandLineArgs
)

// version version
var version string = "1.0.0"

func init() {
	flag.StringVar(&args.DatabaseSourceName, "s", "root:root@tcp(127.0.0.1:3306)/xooooooox?charset=utf8mb4", "database source name")
	flag.StringVar(&args.FilePackageName, "p", "orm", "Package name of file")
	flag.BoolVar(&args.FmtFile, "f", true, "Is fmt go file")
	flag.BoolVar(&args.Json, "j", true, "Whether to add json tag")
	flag.BoolVar(&args.Xorm, "x", false, "Whether to add xorm tag")
	flag.BoolVar(&args.Version, "v", false, "View version")
	flag.StringVar(&args.FileSaveDir, "d", "./", "Address of the saved file")
	flag.StringVar(&args.FileNameSuffix, "i", "_tmp", "Name of file name suffix")
	flag.Parse()
	osArgs := os.Args
	for i := 0; i < len(osArgs); i++ {
		if osArgs[i] == "-v" || osArgs[i] == "--version" {
			fmt.Println(version)
			os.Exit(0)
		}
	}
	if args.DatabaseName == "" {
		args.DatabaseName = args.DatabaseSourceName[strings.Index(args.DatabaseSourceName, "/")+1 : strings.Index(args.DatabaseSourceName, "?")]
	}
	db, err := sql.Open("mysql", args.DatabaseSourceName)
	if err != nil || db == nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	sea.DB = db
}

func main() {
	if err := Write(); err != nil {
		log.Fatal(err)
	}
}

// Write information schema
func Write() error {
	tables, _ := sea.InformationSchemaAllTables(args.DatabaseName)
	lengthTable := len(tables)
	if lengthTable == 0 {
		return errors.New("haven't any table")
	}
	code := Head()
	types := ""
	consts := "const(\n"
	for _, vt := range tables {
		tableName := vt.TableName
		pascalTableName := sea.UnderlineToPascal(tableName)
		consts = fmt.Sprintf("%s\tTab%s = \"%s\" // %s\n", consts, pascalTableName, tableName, vt.TableComment)
		columns, _ := sea.InformationSchemaAllColumns(vt.TableSchema, vt.TableName)
		lengthColumn := len(columns)
		if lengthColumn == 0 {
			continue
		}
		types += fmt.Sprintf("// %s %s %s\n", pascalTableName, tableName, vt.TableComment)
		types += fmt.Sprintf("type %s struct{\n", pascalTableName)
		for _, vc := range columns {
			columnName := vc.ColumnName
			golangType := ColumnDataTypeToGoType(vc.DataType)
			// golang base data type , exist 'unsigned' and 'int' keyword (integer may be unsigned)
			if strings.Index(strings.ToLower(vc.ColumnType), "unsigned") > 0 && strings.Index(strings.ToLower(vc.ColumnType), "int") > 0 {
				golangType = "u" + golangType
			}
			// current column allow null, set type is *type or (sql.NullInt64, sql.NullFloat64, sql.NullString ...), otherwise it causes rows.Scan panic
			if strings.ToUpper(vc.IsNullable) == "YES" {
				golangType = "*" + golangType
			}
			types += fmt.Sprintf("\t%s %s", sea.UnderlineToPascal(columnName), golangType)
			tag := ""
			if args.Json {
				tag = fmt.Sprintf(" json:\"%s\"", vc.ColumnName)
			}
			if args.Xorm {
				tag += fmt.Sprintf(" xorm:\"%s\"", TagXorm(&vc))
			}
			if tag != "" {
				types = fmt.Sprintf("%s `%s`", types, strings.TrimLeft(tag, " "))
			}
			types += fmt.Sprintf(" // %s\n", vc.ColumnComment)
		}
		types += fmt.Sprintf("}\n")
	}
	consts = fmt.Sprintf("%s)\n\n", consts)
	code = fmt.Sprintf("%s%s%s", code, consts, types)
	return WriteFile(args.DatabaseName+args.FileNameSuffix+".go", &code)
}

// Head File head
func Head() string {
	code := "// Copyright (C) xooooooox\n"
	code = fmt.Sprintf("%s// datetime %s\n\n", code, time.Now().Format("2006-01-02 15:04:05"))
	code = fmt.Sprintf("%spackage %s\n\n", code, args.FilePackageName)
	return code
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

// WriteFile write into file
func WriteFile(file string, s *string) error {
	ds := string(filepath.Separator)
	abs, _ := filepath.Abs(args.FileSaveDir)
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
	if args.FmtFile {
		return FmtFile(file)
	}
	return nil
}

// FmtFile fmt file
func FmtFile(file string) error {
	cmd := exec.Command("go", "fmt", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
