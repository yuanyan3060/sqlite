// Package sql 数据库/数据处理相关工具
package sql

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"time"
	"unicode"

	_ "github.com/fumiama/sqlite3" // 引入sqlite

	"github.com/FloatTech/ttl"
)

var (
	ErrNilDB      = errors.New("sqlite: db is not initialized")
	ErrNullResult = errors.New("sqlite: null result")
)

// Sqlite 数据库对象
type Sqlite struct {
	DB        *sql.DB
	DBPath    string
	stmtcache *ttl.Cache[string, *sql.Stmt]
}

// Open 打开数据库
func (db *Sqlite) Open(cachettl time.Duration) (err error) {
	if db.DB == nil {
		database, err := sql.Open("sqlite3", db.DBPath)
		if err != nil {
			return err
		}
		db.DB = database
	}
	if db.stmtcache == nil {
		db.stmtcache = ttl.NewCacheOn(cachettl, [4]func(string, *sql.Stmt){
			nil, nil,
			func(_ string, stmt *sql.Stmt) { _ = stmt.Close() },
			nil,
		})
	}
	return
}

// Close 关闭数据库
func (db *Sqlite) Close() (err error) {
	if db.DB != nil {
		err = db.DB.Close()
		db.DB = nil
		db.stmtcache.Destroy()
		db.stmtcache = nil
	}
	return
}

func wraptable(table string) string {
	first := []rune(table)[0]
	if first < unicode.MaxLatin1 && unicode.IsDigit(first) {
		return "[" + table + "]"
	} else {
		return "'" + table + "'"
	}
}

func (db *Sqlite) compile(q string) (*sql.Stmt, error) {
	stmt := db.stmtcache.Get(q)
	if stmt == nil {
		var err error
		stmt, err = db.DB.Prepare(q)
		if err != nil {
			return nil, err
		}
		db.stmtcache.Set(q, stmt)
	}
	return stmt, nil
}

func (db *Sqlite) mustcompile(q string) *sql.Stmt {
	stmt := db.stmtcache.Get(q)
	if stmt == nil {
		var err error
		stmt, err = db.DB.Prepare(q)
		if err != nil {
			panic(err)
		}
		db.stmtcache.Set(q, stmt)
	}
	return stmt
}

// Create 生成数据库
// 默认结构体的第一个元素为主键
// 返回错误
func (db *Sqlite) Create(table string, objptr interface{}, additional ...string) (err error) {
	if db.DB == nil {
		err = ErrNilDB
		return
	}
	var (
		tags  = tags(objptr)
		kinds = kinds(objptr)
		top   = len(tags) - 1
		cmd   = make([]string, 0, 3*(len(tags)+1))
	)
	cmd = append(cmd, "CREATE TABLE IF NOT EXISTS", wraptable(table), "(")
	if top == 0 {
		cmd = append(cmd, tags[0], kinds[0], "PRIMARY KEY")
		if len(additional) > 0 {
			cmd = append(cmd, ",")
			cmd = append(cmd, strings.Join(additional, ","))
		}
		cmd = append(cmd, ")")
	} else {
		for i := range tags {
			cmd = append(cmd, tags[i], kinds[i])
			switch i {
			default:
				cmd = append(cmd, ",")
			case 0:
				cmd = append(cmd, "PRIMARY KEY,")
			case top:
				if len(additional) > 0 {
					cmd = append(cmd, ",")
					cmd = append(cmd, strings.Join(additional, ","))
				}
				cmd = append(cmd, ")")
			}
		}
	}
	stmt, err := db.compile(strings.Join(cmd, " ") + ";")
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	return
}

// Insert 插入数据集
// 如果 PK 存在会覆盖
// 默认结构体的第一个元素为主键
// 返回错误
func (db *Sqlite) Insert(table string, objptr interface{}) error {
	if db.DB == nil {
		return ErrNilDB
	}
	table = wraptable(table)
	stmt, err := db.compile("SELECT * FROM " + table + " limit 1;")
	if err != nil {
		return err
	}
	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	tags, _ := rows.Columns()
	rows.Close()
	var (
		vals = values(objptr)
		top  = len(tags) - 1
		cmd  = make([]string, 0, 2+4*len(tags))
	)
	cmd = append(cmd, "REPLACE INTO")
	cmd = append(cmd, table)
	if top == 0 {
		cmd = append(cmd, "(")
		cmd = append(cmd, tags[0])
		cmd = append(cmd, ") VALUES ( ? )")
	} else {
		for i := range tags {
			switch i {
			default:
				cmd = append(cmd, tags[i])
				cmd = append(cmd, ",")
			case 0:
				cmd = append(cmd, "(")
				cmd = append(cmd, tags[i])
				cmd = append(cmd, ",")
			case top:
				cmd = append(cmd, tags[i])
				cmd = append(cmd, ")")
			}
		}
		for i := range tags {
			switch i {
			default:
				cmd = append(cmd, "? ,")
			case 0:
				cmd = append(cmd, "VALUES ( ? ,")
			case top:
				cmd = append(cmd, "? )")
			}
		}
	}
	stmt, err = db.compile(strings.Join(cmd, " ") + ";")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(vals...)
	return err
}

// InsertUnique 插入数据集
// 如果 PK 存在会报错
// 默认结构体的第一个元素为主键
// 返回错误
func (db *Sqlite) InsertUnique(table string, objptr interface{}) error {
	if db.DB == nil {
		return ErrNilDB
	}
	table = wraptable(table)
	stmt, err := db.compile("SELECT * FROM " + table + " limit 1;")
	if err != nil {
		return err
	}
	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	tags, _ := rows.Columns()
	rows.Close()
	var (
		vals = values(objptr)
		top  = len(tags) - 1
		cmd  = make([]string, 0, 2+4*len(tags))
	)
	cmd = append(cmd, "INSERT INTO")
	cmd = append(cmd, table)
	if top == 0 {
		cmd = append(cmd, "(")
		cmd = append(cmd, tags[0])
		cmd = append(cmd, ") VALUES ( ? )")
	} else {
		for i := range tags {
			switch i {
			default:
				cmd = append(cmd, tags[i])
				cmd = append(cmd, ",")
			case 0:
				cmd = append(cmd, "(")
				cmd = append(cmd, tags[i])
				cmd = append(cmd, ",")
			case top:
				cmd = append(cmd, tags[i])
				cmd = append(cmd, ")")
			}
		}
		for i := range tags {
			switch i {
			default:
				cmd = append(cmd, "? ,")
			case 0:
				cmd = append(cmd, "VALUES ( ? ,")
			case top:
				cmd = append(cmd, "? )")
			}
		}
	}
	stmt, err = db.compile(strings.Join(cmd, " ") + ";")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(vals...)
	return err
}

// Find 查询数据库，写入最后一条结果到 objptr
// condition 可为"WHERE id = 0"
// 默认字段与结构体元素顺序一致
// 返回错误
func (db *Sqlite) Find(table string, objptr interface{}, condition string) error {
	if db.DB == nil {
		return ErrNilDB
	}
	q := "SELECT * FROM " + wraptable(table) + " " + condition + ";"
	stmt, err := db.compile(q)
	if err != nil {
		return err
	}
	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrNullResult
	}
	err = rows.Scan(addrs(objptr)...)
	for rows.Next() {
		if err != nil {
			return err
		}
		err = rows.Scan(addrs(objptr)...)
	}
	return err
}

// Query 查询数据库，写入最后一条结果到 objptr
// q 为一整条查询语句, 慎用
// 默认字段与结构体元素顺序一致
// 返回错误
func (db *Sqlite) Query(q string, objptr interface{}) error {
	if db.DB == nil {
		return ErrNilDB
	}
	stmt, err := db.compile(q)
	if err != nil {
		return err
	}
	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrNullResult
	}
	err = rows.Scan(addrs(objptr)...)
	for rows.Next() {
		if err != nil {
			return err
		}
		err = rows.Scan(addrs(objptr)...)
	}
	return err
}

// CanFind 查询数据库是否有 condition
// condition 可为"WHERE id = 0"
// 默认字段与结构体元素顺序一致
// 返回错误
func (db *Sqlite) CanFind(table string, condition string) bool {
	if db.DB == nil {
		return false
	}
	q := "SELECT * FROM " + wraptable(table) + " " + condition + ";"
	stmt, err := db.compile(q)
	if err != nil {
		return false
	}
	rows, err := stmt.Query()
	if err != nil {
		return false
	}
	if rows.Err() != nil {
		return false
	}
	defer rows.Close()

	if !rows.Next() {
		return false
	}
	_ = rows.Close()
	return true
}

// CanQuery 查询数据库是否有 q
// q 为一整条查询语句, 慎用
// 默认字段与结构体元素顺序一致
// 返回错误
func (db *Sqlite) CanQuery(q string) bool {
	if db.DB == nil {
		return false
	}
	stmt, err := db.compile(q)
	if err != nil {
		return false
	}
	rows, err := stmt.Query()
	if err != nil {
		return false
	}
	if rows.Err() != nil {
		return false
	}
	defer rows.Close()

	if !rows.Next() {
		return false
	}
	_ = rows.Close()
	return true
}

// FindFor 查询数据库，用函数 f 遍历结果
// condition 可为"WHERE id = 0"
// 默认字段与结构体元素顺序一致
// 返回错误
func (db *Sqlite) FindFor(table string, objptr interface{}, condition string, f func() error) error {
	if db.DB == nil {
		return ErrNilDB
	}
	q := "SELECT * FROM " + wraptable(table) + " " + condition + ";"
	stmt, err := db.compile(q)
	if err != nil {
		return err
	}
	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrNullResult
	}
	err = rows.Scan(addrs(objptr)...)
	if err == nil {
		err = f()
	}
	for rows.Next() {
		if err != nil {
			return err
		}
		err = rows.Scan(addrs(objptr)...)
		if err == nil {
			err = f()
		}
	}
	return err
}

// QueryFor 查询数据库，用函数 f 遍历结果
// q 为一整条查询语句, 慎用
// 默认字段与结构体元素顺序一致
// 返回错误
func (db *Sqlite) QueryFor(q string, objptr interface{}, f func() error) error {
	if db.DB == nil {
		return ErrNilDB
	}
	stmt, err := db.compile(q)
	if err != nil {
		return err
	}
	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrNullResult
	}
	err = rows.Scan(addrs(objptr)...)
	if err == nil {
		err = f()
	}
	for rows.Next() {
		if err != nil {
			return err
		}
		err = rows.Scan(addrs(objptr)...)
		if err == nil {
			err = f()
		}
	}
	return err
}

// Pick 从 table 随机一行
func (db *Sqlite) Pick(table string, objptr interface{}) error {
	if db.DB == nil {
		return ErrNilDB
	}
	return db.Find(table, objptr, "ORDER BY RANDOM() limit 1")
}

// ListTables 列出所有表名
// 返回所有表名+错误
func (db *Sqlite) ListTables() (s []string, err error) {
	if db.DB == nil {
		return nil, ErrNilDB
	}
	rows, err := db.mustcompile("SELECT name FROM sqlite_master where type='table' order by name;").Query()
	if err != nil {
		return
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	defer rows.Close()

	for rows.Next() {
		if err != nil {
			return
		}
		objptr := new(string)
		err = rows.Scan(objptr)
		if err == nil {
			s = append(s, *objptr)
		}
	}
	return
}

// Del 删除数据库表项
// condition 可为"WHERE id = 0"
// 返回错误
func (db *Sqlite) Del(table string, condition string) error {
	if db.DB == nil {
		return ErrNilDB
	}
	q := "DELETE FROM " + wraptable(table) + " " + condition + ";"
	stmt, err := db.compile(q)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	return err
}

// Drop 删除数据库表
func (db *Sqlite) Drop(table string) error {
	if db.DB == nil {
		return ErrNilDB
	}
	q := "DROP TABLE " + wraptable(table) + ";"
	stmt, err := db.compile(q)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	return err
}

// Count 查询数据库行数
// 返回行数以及错误
func (db *Sqlite) Count(table string) (num int, err error) {
	if db.DB == nil {
		return 0, ErrNilDB
	}
	stmt, err := db.compile("SELECT COUNT(1) FROM " + wraptable(table) + ";")
	if err != nil {
		return 0, err
	}
	rows, err := stmt.Query()
	if err != nil {
		return 0, err
	}
	if rows.Err() != nil {
		return 0, rows.Err()
	}
	if rows.Next() {
		err = rows.Scan(&num)
	}
	rows.Close()
	return num, err
}

// tags 反射 返回结构体对象的 tag 数组
func tags(objptr interface{}) (tags []string) {
	elem := reflect.ValueOf(objptr).Elem()
	flen := elem.Type().NumField()
	tags = make([]string, flen)
	for i := 0; i < flen; i++ {
		t := elem.Type().Field(i).Tag.Get("db")
		if t == "" {
			t = elem.Type().Field(i).Tag.Get("json")
			if t == "" {
				t = elem.Type().Field(i).Name
			}
		}
		tags[i] = t
	}
	return
}

// kinds 反射 返回结构体对象的 kinds 数组
func kinds(objptr interface{}) (kinds []string) {
	elem := reflect.ValueOf(objptr).Elem()
	// 判断第一个元素是否为匿名字段
	if elem.Type().Field(0).Anonymous {
		elem = elem.Field(0)
	}
	flen := elem.Type().NumField()
	kinds = make([]string, flen)
	for i := 0; i < flen; i++ {
		typ := elem.Field(i).Type().String()
		switch typ {
		case "bool", "*bool":
			kinds[i] = "BOOLEAN"
		case "int8", "*int8":
			kinds[i] = "TINYINT"
		case "uint8", "byte", "*uint8", "*byte":
			kinds[i] = "UNSIGNED TINYINT"
		case "int16", "*int16":
			kinds[i] = "SMALLINT"
		case "uint16", "*uint16":
			kinds[i] = "UNSIGNED SMALLINT"
		case "int", "*int":
			kinds[i] = "INTEGER"
		case "uint", "*uint":
			kinds[i] = "UNSIGNED INTEGER"
		case "int32", "rune", "*int32", "*rune":
			kinds[i] = "INT"
		case "uint32", "*uint32":
			kinds[i] = "UNSIGNED INT"
		case "int64", "*int64":
			kinds[i] = "BIGINT"
		case "uint64", "*uint64":
			kinds[i] = "UNSIGNED BIGINT"
		case "float32", "*float32":
			kinds[i] = "FLOAT"
		case "float64", "*float64":
			kinds[i] = "DOUBLE"
		case "string", "[]string", "*string", "*[]string":
			kinds[i] = "TEXT"
		default:
			kinds[i] = "BLOB"
		}
		if strings.Contains(typ, "*") || strings.Contains(typ, "[]") {
			kinds[i] += " NULL"
		} else {
			kinds[i] += " NOT NULL"
		}
	}
	return
}

var typstrarr = reflect.SliceOf(reflect.TypeOf(""))

// values 反射 返回结构体对象的 values 数组
func values(objptr interface{}) (values []interface{}) {
	elem := reflect.ValueOf(objptr).Elem()
	flen := elem.Type().NumField()
	values = make([]interface{}, flen)
	for i := 0; i < flen; i++ {
		if elem.Field(i).Type() == typstrarr { // []string
			values[i] = elem.Field(i).Index(0).Interface() // string
			continue
		}
		values[i] = elem.Field(i).Interface()
	}
	return
}

// addrs 反射 返回结构体对象的 addrs 数组
func addrs(objptr interface{}) (addrs []interface{}) {
	elem := reflect.ValueOf(objptr).Elem()
	flen := elem.Type().NumField()
	addrs = make([]interface{}, flen)
	for i := 0; i < flen; i++ {
		if elem.Field(i).Type() == typstrarr { // []string
			s := reflect.ValueOf(make([]string, 1))
			elem.Field(i).Set(s)
			addrs[i] = s.Index(0).Addr().Interface() // string
			continue
		}
		addrs[i] = elem.Field(i).Addr().Interface()
	}
	return
}
