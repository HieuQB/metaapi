//Auto generated with MetaApi https://github.com/exyzzy/metaapi
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
    "strconv"
)

import	"time"


var testDb *sql.DB
var configdb map[string]interface{}
const testDbName = "testtodo"

// ======= helpers

//assumes a configlocaldb.json file as:
//{
//    "Host": "localhost",
//    "Port": "5432",
//    "User": "dbname",
//    "Pass": "dbname",
//    "Name": "dbname",
//    "SSLMode": "disable"
//}
func loadConfig() {
	fmt.Println("  loadConfig")
	file, err := os.Open("configlocaldb.json")
	if err != nil {
		log.Panicln("Cannot open configlocaldb file", err.Error())
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configdb)
	if err != nil {
		log.Panicln("Cannot get local configurationdb from file", err.Error())
	}
}

func createDb(db *sql.DB, dbName string, owner string) (err error) {
	ss := fmt.Sprintf("CREATE DATABASE %s OWNER %s", dbName, owner)
	fmt.Println("  " + ss)
	_, err = db.Exec(ss)
	return
}

func setTzDb(db *sql.DB) (err error) {
	ss := fmt.Sprintf("SET TIME ZONE UTC")
	fmt.Println("  " + ss)
	_, err = db.Exec(ss)
	return
}

func dropDb(db *sql.DB, dbName string) (err error) {
	ss := fmt.Sprintf("DROP DATABASE %s", dbName)
	fmt.Println("  " + ss)
	_, err = db.Exec(ss)
	return
}

func rowExists(db *sql.DB, query string, args ...interface{}) (exists bool, err error) {
	query = fmt.Sprintf("SELECT EXISTS (%s)", query)
	fmt.Println("  " + query)
	err = db.QueryRow(query, args...).Scan(&exists)
	return
}

func tableExists(db *sql.DB, table string) (valid bool, err error) {

	valid, err = rowExists(db, "SELECT 1 FROM pg_tables WHERE tablename = $1", table)
	return
}

func initTestDb() (err error) {
	loadConfig()
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s "+
		"sslmode=%s", configdb["Host"], configdb["Port"], configdb["User"], configdb["Pass"], configdb["SSLMode"])
	testDb, err = sql.Open("postgres", psqlInfo)
	return
}

func TestMain(m *testing.M) {
	//test setup
	err := initTestDb()
	if err != nil {
		log.Panicln("cannot initTestDb ", err.Error())
	}

	err = createDb(testDb, testDbName, configdb["User"].(string))
	if err != nil {
		log.Panicln("cannot CreateDb ", err.Error())
	}

	err = setTzDb(testDb)
	if err != nil {
		log.Panicln("cannot setTzDb ", err.Error())
	}

	//run tests
	exitVal := m.Run()

	//test teardown
	err = dropDb(testDb, testDbName)
	if err != nil {
		log.Panicln("cannot DropDb ", err.Error())
	}
	os.Exit(exitVal)
}

type compareType func(interface{}, interface{}) bool

func noCompare(result, expect interface{}) bool {
	fmt.Printf("  noCompare: %v, %v -  %T, %T \n", result, expect, result, expect)
	return (true)
}

func defaultCompare(result, expect interface{}) bool {
	fmt.Printf("  defaultCompare: %v, %v -  %T, %T \n", result, expect, result, expect)
	return (result == expect)
}

func jsonCompare(result, expect interface{}) bool {
	fmt.Printf("  jsonCompare: %v, %v -  %T, %T \n", result, expect, result, expect)

	//json fields can be any order after db return, so read into map[string]interface and look up
	resultMap := make(map[string]interface{})
	expectMap := make(map[string]interface{})

	if reflect.TypeOf(result).String() == "sql.NullString" {
		err := json.Unmarshal([]byte(result.(sql.NullString).String), &resultMap)
		if err != nil {
			log.Panic(err)
		}
		err = json.Unmarshal([]byte(expect.(sql.NullString).String), &expectMap)
		if err != nil {
			log.Panic(err)
		}
	} else {
		err := json.Unmarshal([]byte(result.(string)), &resultMap)
		if err != nil {
			log.Panic(err)
		}
		err = json.Unmarshal([]byte(expect.(string)), &expectMap)
		if err != nil {
			log.Panic(err)
		}
	}

	for k, v := range expectMap {
		if v != resultMap[k] {
			fmt.Printf("Key: %v, Result: %v, Expect: %v", k, resultMap[k], v)
			return false
		}
	}
	return true
}

	for k, v := range expectMap {
		if v != resultMap[k] {
			fmt.Printf("Key: %v, Result: %v, Expect: %v", k, resultMap[k], v)
			return false
		}
	}
	return true
}

func stringCompare(result, expect interface{}) bool {

	resultJson, err := json.Marshal(result)
	if err != nil {
		log.Panic(err)
	}
	expectJson, err := json.Marshal(expect)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("  stringCompare: %v, %v -  %T, %T \n", string(resultJson), string(expectJson), result, expect)
	return (strings.TrimSpace(string(resultJson)) == strings.TrimSpace(string(expectJson)))
}

//psgl truncs reals at 6 digits
func realCompare(result, expect interface{}) bool {

	fmt.Printf("  realCompare: %v, %v -  %T, %T \n", result, expect, result, expect)

	var resultStr string
	var expectStr string
	if reflect.TypeOf(result).String() == "sql.NullFloat64" {
		resultStr = strconv.FormatFloat(result.(sql.NullFloat64).Float64, 'f', 6, 32)
		expectStr = strconv.FormatFloat(expect.(sql.NullFloat64).Float64, 'f', 6, 32)
	} else {
		resultStr = strconv.FormatFloat(float64(result.(float32)), 'f', 6, 32)
		expectStr = strconv.FormatFloat(float64(expect.(float32)), 'f', 6, 32)
	}
	return (resultStr == expectStr)
}

//iterate through each field of struct and apply the compare function to each field based on compareType map
func equalField(result, expect interface{}, compMap map[string]compareType) error {

	u := reflect.ValueOf(expect)
	v := reflect.ValueOf(result)
	typeOfS := u.Type()

	for i := 0; i < u.NumField(); i++ {

		if !(compMap[typeOfS.Field(i).Name])(v.Field(i).Interface(), u.Field(i).Interface()) {
			return fmt.Errorf("Field: %s, Result: %v, Expect: %v", typeOfS.Field(i).Name, v.Field(i).Interface(), u.Field(i).Interface())
		}
	}
	return nil
}


//table specific 


const todostableName = "todos"

//test data - note: double brackets in test data need space between otherwise are interpreted as template action
var testTodo = [2]Todo{  Id: 1, UpdatedAt: sql.NullTime{time.Now().UTC().Truncate(time.Microsecond), true}, Done: sql.NullBool{true, true}, Title: sql.NullString{"YQe4nNqr1VXQCXyS", true},  Id: 2, UpdatedAt: sql.NullTime{time.Now().UTC().Truncate(time.Microsecond), true}, Done: sql.NullBool{true, true}, Title: sql.NullString{"YQe4nNqr1VXQCXyS", true} }

var updateTodo = Todo Id: 1, UpdatedAt: sql.NullTime{time.Now().UTC().Truncate(time.Microsecond), true}, Done: sql.NullBool{true, true}, Title: sql.NullString{"YQe4nNqr1VXQCXyS", true}

//compare functions
var compareTodos = map[string]compareType{
	"Id": defaultCompare,
	"UpdatedAt": stringCompare,
	"Done": defaultCompare,
	"Title": defaultCompare,

}

func reverseTodos(todos []Todo) (result []Todo) {

	for i := len(todos) - 1; i >= 0; i-- {
		result = append(result, todos[i])
	}
	return
}

// ======= tests: Todo =======

func TestCreateTableTodos(t *testing.T) {
	fmt.Println("==CreateTableTodos")
	err := CreateTableTodos(testDb)
	if err != nil {
		t.Errorf("cannot CreateTableTodos " + err.Error())
	} else {
		fmt.Println("  Done: CreateTableTodos")
	}
	exists, err := tableExists(testDb, "todos")
	if err != nil {
		t.Errorf("cannot tableExists " + err.Error())
	}
	if !exists {
		t.Errorf("tableExists(todos) returned wrong status code: got %v want %v", exists, true)
	} else {
		fmt.Println("  Done: tableExists")
	}
}

func TestCreateTodo(t *testing.T) {
	fmt.Println("==CreateTodo")
	result, err := testTodo[0].CreateTodo(testDb)
	if err != nil {
		t.Errorf("cannot CreateTodo " + err.Error())
	} else {
		fmt.Println("  Done: CreateTodo")
	}
	err = equalField(result, testTodo[0], compareTodos)
	if err != nil {
		t.Errorf("api returned unexpected result. " + err.Error())
	}
}

func TestRetrieveTodo(t *testing.T) {
	fmt.Println("==RetrieveTodo")
	result, err := testTodo[0].RetrieveTodo(testDb)
	if err != nil {
		t.Errorf("cannot RetrieveTodo " + err.Error())
	} else {
		fmt.Println("  Done: RetrieveTodo")
	}
	err = equalField(result, testTodo[0], compareTodos)
	if err != nil {
		t.Errorf("api returned unexpected result. " + err.Error())
	}
}

func TestRetrieveAllTodos(t *testing.T) {
	fmt.Println("==RetrieveAllTodos")
	_, err := testTodo[1].CreateTodo(testDb)
	if err != nil {
		t.Errorf("cannot CreateTodo " + err.Error())
	} else {
		fmt.Println("  Done: CreateTodo")
	}
	result, err := RetrieveAllTodos(testDb)
	if err != nil {
		t.Errorf("cannot RetrieveAllTodos " + err.Error())
	} else {
		fmt.Println("  Done: RetrieveAllTodos")
	}
	//reverse because api is DESC, [:] is slice of all array elements
	expect := reverseTodos(testTodo[:])
	for i, _ := range expect {
		err = equalField(result[i], expect[i], compareTodos)
		if err != nil {
			t.Errorf("api returned unexpected result. " + err.Error())
		}
	}
}


func TestUpdateTodo(t *testing.T) {
	fmt.Println("==UpdateTodo")
	result, err := updateTodo.UpdateTodo(testDb)
	if err != nil {
		t.Errorf("cannot UpdateTodo " + err.Error())
	} else {
		fmt.Println("  Done: UpdateTodo")
	}
	err = equalField(result, updateTodo, compareTodos)
	if err != nil {
		t.Errorf("api returned unexpected result. " + err.Error())
	}
}


//delete all data in reverse order to accommodate foreign keys


func TestDeleteTodo(t *testing.T) {
	fmt.Println("==DeleteTodo")
	err := testTodo[0].DeleteTodo(testDb)
	if err != nil {
		t.Errorf("cannot DeleteTodo " + err.Error())
	} else {
		fmt.Println("  Done: DeleteTodo")
	}
	_, err = testTodo[0].RetrieveTodo(testDb)
	if err == nil {
		t.Errorf("api returned unexpected result: got Row want NoRow")
	} else {
		if err == sql.ErrNoRows {
			fmt.Println("  Done: RetrieveTodo with no result")
		} else {
			t.Errorf("cannot RetrieveTodo " + err.Error())
		}
	}
}

func TestDeleteAllTodos(t *testing.T) {
	fmt.Println("==DeleteAllTodos")
	err := DeleteAllTodos(testDb)
	if err != nil {
		t.Errorf("cannot DeleteAllTodos " + err.Error())
	} else {
		fmt.Println("  Done: DeleteAllTodos")
	}
	result, err := RetrieveAllTodos(testDb)
	if err != nil {
		t.Errorf("cannot RetrieveAllTodos " + err.Error())
	}
	if len(result) > 0 {
		t.Errorf("api returned unexpected result: got Row want NoRow")
	} else {
		fmt.Println("  Done: RetrieveAllTodos with no result")
	}
}
