package main

import (
	"strconv"
	"io/ioutil"
	"database/sql"
	"net/http"
	"fmt"
	"encoding/json"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// NewDbExplorer - Функция получения экземпляра db_explorer
func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	dbExplorer := &DbExplorer{
		db,
		make([]*Table, 0, 0),
	}

	// Тут нужно получить и закешировать список доступных таблиц
	dbExplorer.updateTablesCache()

	return dbExplorer, nil
}

//*************************************************************************
// HTTP Handlers

// DbExplorer - динамическая работа с БД
type DbExplorer struct {
	DB *sql.DB
	Tables []*Table
}

// API
func (h *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s:%s", r.Method, r.URL.Path)
	var paths []string

	// Убираем из Path слэши по краям и разбиваем его на слайс строк
	var path = strings.Trim(r.URL.Path, "/")
	if path != "" {
		paths = strings.Split(strings.Trim(path, "/"), "/")
	} else {
		paths = make([]string, 0, 0)
	}

	var response interface{}
	var err error
	switch  {
	case r.Method == "GET" && len(paths) == 0:
		// GET / - возвращает список все таблиц
		response, err = h.getTables()
		
	case r.Method == "GET" && len(paths) == 1:
		// GET /$table?limit=5&offset=7 - возвращает список из 5 записей (limit) 
		//		начиная с 7-й (offset) из таблицы $table. limit по-умолчанию 5, offset 0
		limit, _ := strconv.Atoi(r.FormValue("limit"))
		offset, _ := strconv.Atoi(r.FormValue("offset"))
		response, err = h.getTableRecords(paths[0], limit, offset)

	case r.Method == "GET" && len(paths) == 2:
		// GET /$table/$id - возвращает информацию о самой записи или 404
		recordID, _ := strconv.Atoi(paths[1])
		response, err = h.getTableRecord(paths[0], recordID)
		
	case r.Method == "PUT" && len(paths) == 1:
		// PUT /$table - создат новую запись
		var data map[string]interface{}
		var body []byte
		body, err = ioutil.ReadAll(r.Body)
		if err == nil {
			json.Unmarshal(body, &data)
			response, err = h.putTableRecord(paths[0], data)
		}
		
	case r.Method == "POST" && len(paths) == 2:
		// POST /$table/$id - обновляет запись
		var data map[string]interface{}
		var body []byte
		body, err = ioutil.ReadAll(r.Body)
		if err == nil {
			json.Unmarshal(body, &data)
			recordID, _ := strconv.Atoi(paths[1])
			response, err = h.postTableRecord(paths[0], recordID, data)
		}

	case r.Method == "DELETE" && len(paths) == 2:
		// DELETE /$table/$id - удаляет запись
		recordID, _ := strconv.Atoi(paths[1])
		response, err = h.deleteTableRecord(paths[0], recordID)

	default:
		err = APIError{http.StatusNotFound, fmt.Errorf("unknown method")}
		return
	}

	getResponse(w, response, err)
}

// GET: /
func (h *DbExplorer) getTables() (map[string]interface{}, error) {
	var result []string = make([]string, 0, len(h.Tables))
	for _, curTable := range h.Tables {
		result = append(result, curTable.Name)
	}

	return map[string]interface{}{
		"tables": result,
	}, nil
}

// GET /$table?limit=5&offset=7
func (h *DbExplorer) getTableRecords(tableName string, limit int, offset int) (map[string]interface{}, error) {
	// Ищем таблицу в Кэше
	var table *Table
	for _, curTable := range h.Tables {
		if curTable.Name == tableName {
			table = curTable
			break
		}
	}
	if table == nil {
		return nil, APIError{http.StatusNotFound,fmt.Errorf("unknown table")}
	}

	// Формируем запрос к БД
	var queryText string
	for _, curField := range table.Fields {
		if (queryText != "") { queryText += "," }
		queryText += fmt.Sprintf("`%s`", curField.Name)
	}
	queryText = fmt.Sprintf("SELECT %s \nFROM `%s`", queryText, table.Name)
	if limit != 0 {
		queryText += fmt.Sprintf("\nLIMIT %v", limit)
	}
	if limit != 0 {
		queryText += fmt.Sprintf("\nOFFSET %v", offset)
	}

	// Выполняем запрос к БД
	var dbResult, err = h.DB.Query(queryText)
	if err != nil {
		return nil, err
	}

	var records []map[string]interface{} = make([]map[string]interface{},0,0)
	for dbResult.Next() {
		// Вытаскиваем в объект из текущей записи
		object, err := h.convertRecordToObject(table.Fields, dbResult)
		if err != nil {
			return nil, err
		}
		records = append(records, object)
	}

	return map[string]interface{}{
		"records": records,
	}, nil
}

// GET: /$table/$id
func (h *DbExplorer) getTableRecord(tableName string, recordID int) (map[string]interface{}, error) {
	// Ищем таблицу в Кэше
	var table *Table
	for _, curTable := range h.Tables {
		if curTable.Name == tableName {
			table = curTable
			break
		}
	}
	if table == nil {
		return nil, APIError{http.StatusNotFound, fmt.Errorf("unknown table")}
	}

	// Формируем запрос к БД
	var queryText string
	for _, curField := range table.Fields {
		if (queryText != "") { queryText += "," }
		queryText += fmt.Sprintf("`%s`", curField.Name)
	}
	queryText = fmt.Sprintf("SELECT %s \nFROM `%s`", queryText, table.Name)

	// Ищем первичный ключ
	for _, curField := range table.Fields {
		if curField.Key == "PRI" {
			queryText += fmt.Sprintf("\nWHERE %s=?", curField.Name)
			break
		}
	}
	
	// Выполняем запрос к БД
	var dbResult, err = h.DB.Query(queryText, recordID)
	if err != nil {
		return nil, err
	}

	var record map[string]interface{}
	for dbResult.Next() {		
		record, err = h.convertRecordToObject(table.Fields, dbResult)
		if err != nil {
			return nil, err
		}
	}

	if record == nil {
		return nil, APIError{http.StatusNotFound, fmt.Errorf("record not found")}
	}

	return map[string]interface{}{
		"record": record,
	}, nil
}

// PUT: /$table
func (h *DbExplorer) putTableRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	// Ищем таблицу в Кэше
	var table *Table
	for _, curTable := range h.Tables {
		if curTable.Name == tableName {
			table = curTable
			break
		}
	}
	if table == nil {
		return nil, APIError{http.StatusNotFound, fmt.Errorf("unknown table")}
	}

	// Формируем запрос к БД
	var queryText string
	var queryTextValues string
	var queryParams = make([]interface{}, 0, 0)
	var PKName string
	
	for _, curField := range table.Fields {
		if curField.Key == "PRI" {
			PKName = curField.Name
		}
		if curField.Extra == "auto_increment" {
			// Пропускаем автоинрементные поля
			continue
		}
		// Вытаскиваем значение из входной структуры
		curFieldValue, curFieldExist := data[curField.Name]
		if !curFieldExist {
			curFieldValue = curField.defaultValue()
		}
		queryParams = append(queryParams, curFieldValue)

		if (queryText != "") { queryText += "," }
		if (queryTextValues != "") { queryTextValues += "," }
		queryText += fmt.Sprintf("`%s`", curField.Name)
		queryTextValues += "?"
	}
	queryText = fmt.Sprintf("INSERT INTO `%s` \n(%s) \nVALUES (%s)", table.Name, queryText, queryTextValues)

	// Выполняем запрос к БД
	var dbResult, err = h.DB.Exec(queryText, queryParams...)
	if err != nil {
		return nil, err
	}

	lastID, err := dbResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	var response = make(map[string]interface{})
	response[PKName] = lastID
	return response, nil
}

// POST: /$table/$id
func (h *DbExplorer) postTableRecord(tableName string, recordID int, data map[string]interface{}) (map[string]interface{}, error) {
	// Ищем таблицу в Кэше
	// TODO вынести в отдельную функцию
	var table *Table
	for _, curTable := range h.Tables {
		if curTable.Name == tableName {
			table = curTable
			break
		}
	}
	if table == nil {
		return nil, APIError{http.StatusNotFound, fmt.Errorf("unknown table")}
	}

	// Формируем запрос к БД
	var queryText string
	var queryParams = make([]interface{}, 0, 0)
	
	for _, curField := range table.Fields {
		// Вытаскиваем значение из входной структуры
		curFieldValue, curFieldExist := data[curField.Name]
		if !curFieldExist {
			continue
		}
		if curField.Key == "PRI" {
			// Первичный ключ обновлять нельзя!
			return nil, APIError{http.StatusBadRequest, fmt.Errorf("field %s have invalid type", curField.Name)}
		}
		if curField.Extra == "auto_increment" {
			// Пропускаем автоинрементные поля
			continue
		}
		err := curField.validateFieldType(curFieldValue)
		if err != nil {
			return nil, APIError{http.StatusBadRequest, err}
		}
		
		queryParams = append(queryParams, curFieldValue)

		if (queryText != "") { queryText += "," }
		queryText += fmt.Sprintf("`%s`=?", curField.Name)
	}
	queryText = fmt.Sprintf("UPDATE `%s` \nSET %s", table.Name, queryText)
	
	// Ищем первичный ключ
	for _, curField := range table.Fields {
		if curField.Key == "PRI" {
			queryText += fmt.Sprintf("\nWHERE %s=?", curField.Name)
			queryParams = append(queryParams, recordID)
			break
		}
	}

	// Выполняем запрос к БД
	var dbResult, err = h.DB.Exec(queryText, queryParams...)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := dbResult.RowsAffected()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"updated": rowsAffected,
	}, nil
}

// POST: /$table/$id
func (h *DbExplorer) deleteTableRecord(tableName string, recordID int) (map[string]interface{}, error) {
	// Ищем таблицу в Кэше
	var table *Table
	for _, curTable := range h.Tables {
		if curTable.Name == tableName {
			table = curTable
			break
		}
	}
	if table == nil {
		return nil, APIError{http.StatusNotFound, fmt.Errorf("unknown table")}
	}

	// Формируем запрос к БД
	var queryText = fmt.Sprintf("DELETE FROM `%s`", table.Name)

	// Ищем первичный ключ
	for _, curField := range table.Fields {
		if curField.Key == "PRI" {
			queryText += fmt.Sprintf("\nWHERE %s=?", curField.Name)
			break
		}
	}

	// Выполняем запрос к БД
	var dbResult, err = h.DB.Exec(queryText, recordID)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := dbResult.RowsAffected()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"deleted": rowsAffected,
	}, nil
}

//*************************************************************************
// Работа с БД

// Table - Таблица БД
type Table struct {
	Name string
	Fields []*TableField
}

// TableField - колонки таблицы БД
type TableField struct {
	Name string
	Type string
	Collation sql.NullString
	Nullable string
	Key string
	Default sql.NullString
	Extra string
	Priveleges string
	Comment string
}

func (tf TableField) validateFieldType(currField interface{}) error {
	// Проверка на NULL
	if currField == nil {
		if tf.Nullable != "YES"{
			return fmt.Errorf("field %s have invalid type", tf.Name)
		}
		return nil
	}
	// Проверка на тип
	switch currField.(type) {
	case int:
		if tf.Type == "int" {
			return nil
		}
	case string:
		if strings.Index(tf.Type, "varchar") >= 0 || tf.Type == "text" {
			return nil
		}
	case float64:
		if tf.Type == "double" {
			return nil
		}
	}

	return fmt.Errorf("field %s have invalid type", tf.Name)
}

func (tf TableField) defaultValue() interface{} {
	var v interface{}

	switch {
	case tf.Type == "text":
		v = new(string)
	case tf.Type == "int" && tf.Nullable == "YES":
		v = nil
	case tf.Type == "int" && tf.Nullable != "YES":
		v = new(int)
	case strings.Contains(tf.Type, "varchar") && tf.Nullable == "YES":
		v = nil
	case strings.Contains(tf.Type, "varchar") && tf.Nullable != "YES":
		v = new(string)
	default:
		v = new(interface{})
	}

	return v
}

func (h *DbExplorer) updateTablesCache() {
	// Получаем список таблиц БД
	var dbTables, err = h.DB.Query("SHOW TABLES;")
	if err != nil {
		panic(fmt.Sprintf("Не удалось получить список таблиц: %s" , err.Error()))
	}

	for dbTables.Next() {
		var table = &Table{
			"",
			make([]*TableField, 0, 0),
		}
		err = dbTables.Scan(&table.Name)
		if err != nil {
			panic(fmt.Sprintf("Не удалось получить список таблиц: %s" , err.Error()))
		}
		h.Tables = append(h.Tables, table)
	}

	// Собираем информацию по колонкам всех таблиц 
	for _, curTable := range h.Tables {
		var dbFields, err = h.DB.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", curTable.Name))
		if err != nil {
			panic(fmt.Sprintf("Не удалось получить колонки таблицы %s: %s", curTable.Name, err.Error()))
		}

		for dbFields.Next() {
			var field = &TableField{}
			err = dbFields.Scan(&field.Name, &field.Type, &field.Collation, &field.Nullable, &field.Key, &field.Default, &field.Extra, &field.Priveleges, &field.Comment)
			if err != nil {
				panic(fmt.Sprintf("Не удалось получить колонки таблицы %s: %s", curTable.Name, err.Error()))
			}
			curTable.Fields = append(curTable.Fields, field)
		}
	}
}

// convertRecordToObject - Конвертация строки результата запроса в мапу
func (h *DbExplorer) convertRecordToObject(fields []*TableField, row *sql.Rows) (map[string]interface{}, error) {
	// Объект, в который распакует строку результата запроса
	object := map[string]interface{}{}
	var err error

	// Слайс подготовим для вытягивания значений из результатов запроса
	values := make([]interface{}, len(fields))
	for i, column := range fields {
		var v interface{}

		// По типу колонки кладем в переменную значение нужного типа
		switch {
		case column.Type == "text":
			v = new(string)
		case column.Type == "int" && column.Nullable == "YES":
			v = new(sql.NullInt32)
		case column.Type == "int" && column.Nullable != "YES":
			v = new(int)
		case strings.Contains(column.Type, "varchar") && column.Nullable == "YES":
			v = new(sql.NullString)
		case strings.Contains(column.Type, "varchar") && column.Nullable != "YES":
			v = new(string)
		default:
			v = new(interface{})
		}

		//object[column.Name] = v
		values[i] = v
	}

	err = row.Scan(values...)
	if err != nil {
		return nil, err
	}

	for i, column := range fields {
		// По типу колонки кладем в переменную значение нужного типа
		switch {
		case column.Type == "int" && column.Nullable == "YES":
			v := values[i].(*sql.NullInt32)
			if v.Valid {
				object[column.Name] = v.Int32
			} else {
				object[column.Name] = nil
			}
		case strings.Contains(column.Type, "varchar") && column.Nullable == "YES":
			v := values[i].(*sql.NullString)
			if v.Valid {
				object[column.Name] = v.String
			} else {
				object[column.Name] = nil
			}
		default:
			object[column.Name] = values[i]
		}
	}

	return object, nil
}

//*************************************************************************
// Общие

// APIError - специальный тип для пользовательских ошибок
type APIError struct {
	HTTPStatus int
	Err error
}

func (ae APIError) Error() (string) {
	return ae.Err.Error()
}

// Функция формирования ответа на http-запросы
func getResponse(w http.ResponseWriter, response interface{}, err interface{}) {
	var body map[string]interface{}
	var status int
	if err != nil {
		switch v := err.(type) {
		case APIError:
			status = v.HTTPStatus
			body = map[string]interface{}{
				"error": v.Err.Error(),
			}
		case error:
			status = http.StatusInternalServerError
			body = map[string]interface{}{
				"error": v.Error(),
			}
		default:
			status = http.StatusInternalServerError
			body = map[string]interface{}{
				"error": "unknown error",
			}
		}
	} else {
		status = http.StatusOK
		body = map[string]interface{}{
			"response": response,
		}
	}

	bodyJSON, errJSON := json.Marshal(body)
	if errJSON != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, errJSON.Error())
		return
	}

	w.WriteHeader(status)
	fmt.Fprintln(w, string(bodyJSON))
}