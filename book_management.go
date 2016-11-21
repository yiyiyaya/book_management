package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	//"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/asiainfoLDP/datahub_commons/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
)

const BookTableCreateSql = `
create table IF NOT EXISTS book
(   
    book_id  BIGINT NOT NULL AUTO_INCREMENT,
    name char(100),
    page int(20),
    author char(100),
    PRIMARY KEY (book_id)
)DEFAULT CHARSET=UTF8;
`

func CreateBookTable(db *sql.DB) {
	db.Exec(BookTableCreateSql)
}

func connectDB() {

	DB_ADDR := os.Getenv("MYSQL_ADDR")
	DB_PORT := os.Getenv("MYSQL_PORT")
	DB_DATABASE := os.Getenv("MYSQL_DATABASE")
	DB_USER := os.Getenv("MYSQL_USER")
	DB_PASSWORD := os.Getenv("MYSQL_PASSWORD")
	DB_URL := fmt.Sprintf(`%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true`,
		DB_USER, DB_PASSWORD, DB_ADDR, DB_PORT, DB_DATABASE)

	log.DefaultlLogger().Info("connect to ", DB_URL)

	db, err := sql.Open("mysql", DB_URL) // ! here, err is always nil, db is never nil.
	if err == nil {
		err = db.Ping()
	}

	if err != nil {
		log.DefaultlLogger().Fatal("error:", err)
	} else {
		setDB(db)
		CreateBookTable(db)
	}
}

var (
	dbInstance *sql.DB
	dbMutex    sync.Mutex
)

func getDB() *sql.DB {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	return dbInstance
}

func setDB(db *sql.DB) {
	dbMutex.Lock()
	dbInstance = db
	dbMutex.Unlock()
}

//======================================================
//
//======================================================

var Json_ErrorBuildingJson []byte

func getJsonBuildingErrorJson() []byte {
	if Json_ErrorBuildingJson == nil {
		Json_ErrorBuildingJson = []byte(`{"msg": "json error"}`)
	}

	return Json_ErrorBuildingJson
}

type Result struct {
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

/*
{
    "mag": "ok",
    "data": {
        "title": "abc",
        "author": "zhang3",
        "pages": 123
    }
}
*/

// if data only has one item, then the item key will be ignored.
func JsonResult(w http.ResponseWriter, statusCode int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	result := Result{Msg: msg, Data: data}
	jsondata, err := json.MarshalIndent(&result, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(getJsonBuildingErrorJson()))
	} else {
		w.WriteHeader(statusCode)
		w.Write(jsondata)
	}
}

type QueryListResult struct {
	Total   int64       `json:"total"`
	Results interface{} `json:"results"`
}

func newQueryListResult(count int64, results interface{}) *QueryListResult {
	return &QueryListResult{Total: count, Results: results}
}

func GetRequestData(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
func ParseRequestJsonInto(r *http.Request, into interface{}) error {
	data, err := GetRequestData(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, into)
}

type Book struct {
	Book_id int64  `json:"book_id,omitempty`
	Name    string `json:"name,omitempty"`
	Page    int    `json:"page,omitempty`
	Author  string `json:"author,omitempty`
}

//========================================

/*
POST /book/v1/books

{
    "title": "abc",
    "author": "zhang3",
    "pages": 123
}

curl -X POST http://localhost:8002/book/v1/books \
  -d '{"name": "abc", "author": "zhang3", "page": 123}'
*/

func CreateBook(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, "db not inited", nil)
		return
	}
	sqlstr := `insert into book(name,page,author) values( ?, ?,?)`
	book := Book{}
	err := ParseRequestJsonInto(r, &book)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	result, err := db.Exec(sqlstr, book.Name, book.Page, book.Author)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	book_id, err := result.LastInsertId()
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	JsonResult(w, http.StatusOK, "", book_id)
}

//========================================

/*
PUT /book/v1/books

{
    "title": "abc",
    "author": "zhang3",
    "pages": 123
}

curl -X PUT http://localhost:8002/book/v1/books/4 \
  -d '{"name": "minmin", "author": "lxl", "page": 0}'
*/
func UpdateBook(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, "db not inited", nil)
		return
	}
	id_str := params.ByName("id")
	book_id, err := strconv.ParseInt(id_str, 10, 64)
	if err != nil {
		JsonResult(w, http.StatusBadRequest, "转化错误", nil)
		return
	}

	sqlstr := `update book set name=?,page=?,author=? where book_id=?`
	book := Book{}
	err = ParseRequestJsonInto(r, &book)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	_, err = db.Exec(sqlstr, book.Name, book.Page, book.Author, book_id)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	JsonResult(w, http.StatusOK, "", nil)

}

/*
  curl -X GET  http://localhost:8002/book/v1/books/5 \
  -d '{}'

*/
func GetBook(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, "db not inited", nil)
		return
	}
	id_str := params.ByName("id")
	book_id, err := strconv.ParseInt(id_str, 10, 64)
	if err != nil {
		JsonResult(w, http.StatusBadRequest, "转化错误", nil)
		return
	}

	sqlstr := `select name,author,page from book where book_id=?`

	rows, err := db.Query(sqlstr, book_id)
	if err != nil {
		log.DefaultlLogger().Errorf("查找失败")
		JsonResult(w, http.StatusBadRequest, "查找失败", nil)
		return
	}
	defer rows.Close()
	var book *Book
	for rows.Next() {
		book = &Book{}
		err := rows.Scan(
			&book.Name,
			&book.Author,
			&book.Page)
		if err != nil {
			log.DefaultlLogger().Errorf("查找失败2")
			JsonResult(w, http.StatusBadRequest, "查找失败2", nil)
			return
		}
	}
	if err := rows.Err(); err != nil {
		log.DefaultlLogger().Errorf("查找失败3")
		JsonResult(w, http.StatusBadRequest, "查找失败3", nil)
		return
	}
	if book == nil {
		log.DefaultlLogger().Errorf("没找到")
		JsonResult(w, http.StatusNotFound, "没找到", nil)
		return
	}
	/*jsondata, err := json.MarshalIndent(&book, "", "  ")
	if err != nil {
		log.DefaultlLogger().Errorf("转换失败")
		JsonResult(w, http.StatusBadRequest, "转换失败", nil)
		return
	}
	log.DefaultlLogger().Info(string(jsondata))*/
	book.Book_id = book_id
	JsonResult(w, http.StatusOK, "", book)
}
func QueryBooks(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, "db not inited", nil)
		return
	}

	sqlstr := `select name,author,page from book`

	rows, err := db.Query(sqlstr)
	if err != nil {
		log.DefaultlLogger().Errorf("查找失败")
		JsonResult(w, http.StatusBadRequest, "查找失败", nil)
		return
	}
	defer rows.Close()
	/*type BooksSlice struct {
		book []Book
	}
	var b *BooksSlice
	b.book = append(b.book, Book{Book_id, Name, Author, Page})*/

	//book := make([]*string, 1024)
	var books []*Book
	for rows.Next() {
		book := &Book{}
		err := rows.Scan(
			&book.Name,
			&book.Author,
			&book.Page)

		if err != nil {
			log.DefaultlLogger().Errorf("查找失败2")
			JsonResult(w, http.StatusBadRequest, "查找失败2", nil)
			return
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		log.DefaultlLogger().Errorf("查找失败3")
		JsonResult(w, http.StatusBadRequest, "查找失败3", nil)
		return
	}

	log.DefaultlLogger().Info("执行到了")
	/*	jsondata, err := json.MarshalIndent(&book, "", " ")
		if err != nil {
			log.DefaultlLogger().Errorf("转换失败")
			JsonResult(w, http.StatusBadRequest, "转换失败", nil)
			return
		}
		log.DefaultlLogger().Info(string(jsondata))*/
	JsonResult(w, http.StatusOK, "", books)
}
func DeleteBook(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, "db not inited", nil)
		return
	}
	id_str := params.ByName("id")
	book_id, err := strconv.ParseInt(id_str, 10, 64)
	if err != nil {
		JsonResult(w, http.StatusBadRequest, "转化错误", nil)
		return
	}

	sqlstr := `delete from book where book_id=?`
	print("执行到了")
	log.DefaultlLogger().Info("jddddddddd")
	//book := Book{}
	//err = ParseRequestJsonInto(r, &book)
	//if err != nil {
	//	JsonResult(w, http.StatusInternalServerError, err.Error(), nil)
	//	return
	//}
	_, err = db.Exec(sqlstr, book_id)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	JsonResult(w, http.StatusOK, "", nil)
}
func main() {
	connectDB()
	router := httprouter.New()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.POST("/book/v1/books", CreateBook)
	router.PUT("/book/v1/books/:id", UpdateBook)
	router.GET("/book/v1/books/:id", GetBook)
	router.GET("/book/v1/books", QueryBooks)
	router.DELETE("/book/v1/books/:id", DeleteBook)
	//-------------------------------------------------------------------
	err := http.ListenAndServe(":9091", router) //设置监听的端口
	if err != nil {

		log.DefaultlLogger().Fatal("ListenAndServe:	", err)
	}
}
