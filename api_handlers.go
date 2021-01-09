
// !!! ЭТОТ МОДУЛЬ СГЕНЕРИРОВАН АВТОМАТИЧЕСКИ!
// 	go build ./handlers_gen/codegen.go
// 	./codegen.exe api.go api_handlers.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)


// MyApi
func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
			case "/user/profile": h.handlerMyApiProfile(w, r)
		case "/user/create": h.handlerMyApiCreate(w, r)

	default:
		getResponse(w, ApiError{http.StatusNotFound, fmt.Errorf("unknown method")})
	}
}

// MyApi
func (h *MyApi) handlerMyApiProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	
	// Получаем параметры
	var params ProfileParams
	var err error
	

	// Login
	params.Login = r.FormValue("login")
	// apivalidator: required
	if r.FormValue("login") == "" {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")})
		return
	}
	user, err := h.Profile(ctx, params)
	if err != nil {
		getResponse(w, err)
	} else {
		getResponse(w, user)
	}
}

// MyApi
func (h *MyApi) handlerMyApiCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// "method": "POST"
	if r.Method != "POST" {
		getResponse(w, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")})
		return
	}
	
	// "auth": true
	if r.Header.Get("X-Auth") != "100500" {
		getResponse(w, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")})
		return
	}
	// Получаем параметры
	var params CreateParams
	var err error
	

	// Login
	params.Login = r.FormValue("login")
	// apivalidator: required
	if r.FormValue("login") == "" {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")})
		return
	}
	// apivalidator: min=10
	if utf8.RuneCountInString(params.Login) < 10 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("login len must be >= 10")})
		return
	}

	// Name
	params.Name = r.FormValue("full_name")

	// Status
	params.Status = r.FormValue("status")
	// apivalidator: default=...
	if r.FormValue("status") == "" {
		params.Status = "user"
	}
	// apivalidator: enum=|user|moderator|admin|
	if strings.Index("|user|moderator|admin|", "|"+params.Status+"|") < 0 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("status must be one of [user, moderator, admin]")})
		return
	}

	// Age
	params.Age, err = strconv.Atoi(r.FormValue("age"))
	if err != nil {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("age must be int")})
		return
	}
	// apivalidator: min=0
	if params.Age < 0 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("age must be >= 0")})
		return
	}
	// apivalidator: max=128
	if params.Age > 128 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("age must be <= 128")})
		return
	}
	user, err := h.Create(ctx, params)
	if err != nil {
		getResponse(w, err)
	} else {
		getResponse(w, user)
	}
}

// OtherApi
func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
			case "/user/create": h.handlerOtherApiCreate(w, r)

	default:
		getResponse(w, ApiError{http.StatusNotFound, fmt.Errorf("unknown method")})
	}
}

// OtherApi
func (h *OtherApi) handlerOtherApiCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// "method": "POST"
	if r.Method != "POST" {
		getResponse(w, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")})
		return
	}
	
	// "auth": true
	if r.Header.Get("X-Auth") != "100500" {
		getResponse(w, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")})
		return
	}
	// Получаем параметры
	var params OtherCreateParams
	var err error
	

	// Username
	params.Username = r.FormValue("username")
	// apivalidator: required
	if r.FormValue("username") == "" {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("username must me not empty")})
		return
	}
	// apivalidator: min=3
	if utf8.RuneCountInString(params.Username) < 3 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("username len must be >= 3")})
		return
	}

	// Name
	params.Name = r.FormValue("account_name")

	// Class
	params.Class = r.FormValue("class")
	// apivalidator: default=...
	if r.FormValue("class") == "" {
		params.Class = "warrior"
	}
	// apivalidator: enum=|warrior|sorcerer|rouge|
	if strings.Index("|warrior|sorcerer|rouge|", "|"+params.Class+"|") < 0 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("class must be one of [warrior, sorcerer, rouge]")})
		return
	}

	// Level
	params.Level, err = strconv.Atoi(r.FormValue("level"))
	if err != nil {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("level must be int")})
		return
	}
	// apivalidator: min=1
	if params.Level < 1 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("level must be >= 1")})
		return
	}
	// apivalidator: max=50
	if params.Level > 50 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("level must be <= 50")})
		return
	}
	user, err := h.Create(ctx, params)
	if err != nil {
		getResponse(w, err)
	} else {
		getResponse(w, user)
	}
}


func getResponse(w http.ResponseWriter, response interface{}) {
	var body map[string]interface{}
	var status int
	switch v := response.(type) {
	case ApiError:
		{
			status = v.HTTPStatus
			body = map[string]interface{}{
				"error": v.Err.Error(),
			}
		}
	case error:
		{
			status = http.StatusInternalServerError
			body = map[string]interface{}{
				"error": v.Error(),
			}
		}
	default:
		{
			status = http.StatusOK
			body = map[string]interface{}{
				"error":    "",
				"response": v,
			}
		}
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	fmt.Fprintln(w, string(bodyJSON))
}
