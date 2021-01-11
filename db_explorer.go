package main

import (
	"database/sql"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

// Функция получения экземпляра db_explorer
func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	dbExplorer := &DbExplorer{db}

	// TODO: тут нужно получить и закешировать список доступных таблиц

	return dbExplorer, nil
}

// DbExplorer - динамическая работа с БД
type DbExplorer struct {
	DBConnection *sql.DB
}

// API
func (h *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		//h.handlerMyApiProfile(w, r)
	case "/user/create":
		//h.handlerMyApiCreate(w, r)

	default:
		//getResponse(w, ApiError{http.StatusNotFound, fmt.Errorf("unknown method")})
	}
}
