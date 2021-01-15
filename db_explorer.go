package main

import (
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

	var status int = http.StatusOK
	var response interface{}
	var err error
	switch  {
	case r.Method == "GET" && len(paths) == 0:
		// GET / - возвращает список все таблиц
		response, err = h.getTables()
	case r.Method == "GET" && len(paths) == 1:
		// GET /$table?limit=5&offset=7 - возвращает список из 5 записей (limit) 
		//		начиная с 7-й (offset) из таблицы $table. limit по-умолчанию 5, offset 0
		response, err = h.getTableRecords(paths[0], 0, 0)
	case r.Method == "GET" && len(paths) == 2:
		// GET /$table/$id - возвращает информацию о самой записи или 404
		w.WriteHeader(http.StatusNotAcceptable)
		return
	case r.Method == "PUT" && len(paths) == 1:
		// PUT /$table - создат новую запись

		// TODO
	case r.Method == "POST" && len(paths) == 2:
		// POST /$table/$id - обновляет запись

		// TODO
	case r.Method == "DELETE" && len(paths) == 2:
		// DELETE /$table/$id - удаляет запись

		// TODO
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err != nil {
		status = http.StatusInternalServerError
		response = map[string]interface{}{
			"error": err.Error(),
		}
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	fmt.Fprintln(w, string(responseJSON))
	return
}

func (h *DbExplorer) getTables() (string, error) {
	var result []string = make([]string, 0, len(h.Tables))
	for _, curTable := range h.Tables {
		result = append(result, curTable.Name)
	}

	bodyJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(bodyJSON), nil
}

func (h *DbExplorer) getTableRecords(tableName string, limit int, offset int) (map[string]interface{}, error) {
	//$table?limit=5&offset=7
	// Ищем таблицу в Кэше
	var table *Table
	for _, table = range h.Tables {
		if table.Name == tableName {
			break
		}
	}
	if table == nil {
		return nil, fmt.Errorf("unknown table")
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