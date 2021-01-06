package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		h.handlerProfile(w, r)
	case "/user/create":
		h.handlerCreate(w, r)
	default:
		getResponse(w, ApiError{http.StatusNotFound, fmt.Errorf("unknown method")})
	}
}

func (h *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fieldName := "login"
	params := ProfileParams{r.FormValue(fieldName)}
	if params.Login == "" {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf(fieldName + " must me not empty")})
		return
	}

	user, err := h.Profile(ctx, params)
	if err != nil {
		getResponse(w, err)
	} else {
		getResponse(w, user)
	}
}

func (h *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
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

	// Получаем параметр
	params.Login = r.FormValue("login")
	// apivalidator: required
	if params.Login == "" {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")})
		return
	}
	// apivalidator: min=10
	if utf8.RuneCountInString(params.Login) < 10 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("login len must be >= 10")})
		return
	}

	params.Name = r.FormValue("full_name")

	params.Status = r.FormValue("status")
	// apivalidator: default=user
	if params.Status == "" {
		params.Status = "user"
	}
	// apivalidator: enum=user|moderator|admin
	if strings.Index("|user|moderator|admin|", "|"+params.Status+"|") < 0 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("status must be one of [user, moderator, admin]")})
		return
	}

	// Конвертируем в Int
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

	// Запускаем процедуру
	user, err := h.Create(ctx, params)
	if err != nil {
		getResponse(w, err)
	} else {
		getResponse(w, user)
	}
}

func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		//h.handlerUserProfile(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
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
