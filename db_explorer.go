package main

import (
	"net/http"
)

func NewDbExplorer(db) DbExplorer, error {
	return DbExplorer{db}
}

type DbExplorer struct {
	DB *DB
}

// API
func (h *DbExplorer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		//h.handlerMyApiProfile(w, r)
	case "/user/create":
		//h.handlerMyApiCreate(w, r)

	default:
		//getResponse(w, ApiError{http.StatusNotFound, fmt.Errorf("unknown method")})
	}
}
