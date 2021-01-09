package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

type tpl struct {
	FieldName string
}

var (
	headerTpl = template.Must(template.New("headerTpl").Parse(`
// !!! ЭТОТ МОДУЛЬ СГЕНЕРИРОВАН АВТОМАТИЧЕСКИ!
// 	go build ./handlers_gen/codegen.go
// 	./codegen.exe api.go api_handlers.go
package {{.FieldName}}

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)
`))

	serveHTTPTpl = template.Must(template.New("serveHTTPTpl").Parse(`
// {{.API}}
func (h *{{.API}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	{{.Handlers}}
	default:
		getResponse(w, ApiError{http.StatusNotFound, fmt.Errorf("unknown method")})
	}
}
`))

	handlerTpl = template.Must(template.New("handlerTpl").Parse(`
// {{.API}}
func (h *{{.API}}) {{.HandlerName}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	{{if (ne .Method "")}}
	// "method": "POST"
	if r.Method != "POST" {
		getResponse(w, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")})
		return
	}{{end}}
	{{if .Auth}}
	// "auth": true
	if r.Header.Get("X-Auth") != "100500" {
		getResponse(w, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")})
		return
	}{{end}}
	// Получаем параметры
	var params {{.ParamsName}}
	var err error
	{{.Params}}
	user, err := h.{{.FuncName}}(ctx, params)
	if err != nil {
		getResponse(w, err)
	} else {
		getResponse(w, user)
	}
}
`))

	funcGetResponse = template.Must(template.New("funcGetResponse").Parse(`
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
`))

	paramRequiredTpl = template.Must(template.New("paramRequiredTpl").Parse(`
	// apivalidator: required
	if r.FormValue("{{.Paramname}}") == "" {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("{{.Paramname}} must me not empty")})
		return
	}`))

	paramStringTpl = template.Must(template.New("paramStringTpl").Parse(`
	params.{{.Name}} = r.FormValue("{{.Paramname}}")`))

	paramIntTpl = template.Must(template.New("paramIntTpl").Parse(`
	params.{{.Name}}, err = strconv.Atoi(r.FormValue("{{.Paramname}}"))
	if err != nil {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("{{.Paramname}} must be int")})
		return
	}`))

	paramDefaultTpl = template.Must(template.New("paramDefaultTpl").Parse(`
	// apivalidator: default=...
	if r.FormValue("{{.Paramname}}") == "" {
		params.{{.Name}} = "{{.Default}}"
	}`))

	paramEnumTpl = template.Must(template.New("paramEnumTpl").Parse(`
	// apivalidator: enum={{.Enum}}
	if strings.Index("{{.Enum}}", "|"+params.{{.Name}}+"|") < 0 {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("{{.Paramname}} must be one of [{{.EnumErr}}]")})
		return
	}`))

	paramMinStringTpl = template.Must(template.New("paramMinStringTpl").Parse(`
	// apivalidator: min={{.Min}}
	if utf8.RuneCountInString(params.{{.Name}}) < {{.Min}} {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("{{.Paramname}} len must be >= {{.Min}}")})
		return
	}`))

	paramMaxStringTpl = template.Must(template.New("paramMaxStringTpl").Parse(`
	// apivalidator: max={{.Max}}
	if utf8.RuneCountInString(params.{{.Name}}) < {{.Max}} {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("{{.Paramname}} len must be <= {{.Max}}")})
		return
	}`))

	paramMinIntTpl = template.Must(template.New("paramMinIntTpl").Parse(`
	// apivalidator: min={{.Min}}
	if params.{{.Name}} < {{.Min}} {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("{{.Paramname}} must be >= {{.Min}}")})
		return
	}`))

	paramMaxIntTpl = template.Must(template.New("paramMaxIntTpl").Parse(`
	// apivalidator: max={{.Max}}
	if params.{{.Name}} > {{.Max}} {
		getResponse(w, ApiError{http.StatusBadRequest, fmt.Errorf("{{.Paramname}} must be <= {{.Max}}")})
		return
	}`))
)

/*
type api struct {
	Name    string
	Handlers map[string]apiMethod
}
*/

type apiMethod struct {
	Name        string `json:"-"`
	APIName     string `json:"-"`
	ParamsName  string `json:"-"`
	HandlerName string `json:"-"`
	URL         string `json:"url"`
	Auth        bool   `json:"auth"`
	Method      string `json:"method"`
}

type apiMethodParams struct {
	ParamsName string
	Name       string
	Type       string
	Required   bool
	Paramname  string
	Enum       string
	Default    string
	HasMin     bool
	HasMax     bool
	Min        int
	Max        int
}

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	//node, err := parser.ParseFile(fset, "../api.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])
	//out, _ := os.Create("api_handlers.go")

	apis, apiMethods, apiMethodsParams := prepareCodegen(node)

	headerTpl.Execute(out, tpl{node.Name.Name})
	fmt.Fprintln(out) // empty line

	codegenAPI(out, apis, apiMethods, apiMethodsParams)

	funcGetResponse.Execute(out, tpl{node.Name.Name})
}

func prepareCodegen(node *ast.File) (map[string]string, map[string]*apiMethod, map[string][]*apiMethodParams) {

	apis := make(map[string]string, 0)                         // все API, для которых нашли обработчики
	apiMethods := make(map[string]*apiMethod, 0)               // методы, помеченные "apigen:api"
	apiMethodsParams := make(map[string][]*apiMethodParams, 0) // Структуры, содержащие описания "apivalidator"

	for _, decl := range node.Decls {
		switch decl := decl.(type) {
		// ОБРАБОТКА ФУНКЦИЙ
		case *ast.FuncDecl:
			// Проверяем наличие apigen:api в комментарии и собираем комментарий в строку
			needCodegen := false
			funcComment := ""
			if decl.Doc != nil {
				for _, comment := range decl.Doc.List {
					needCodegen = needCodegen || strings.HasPrefix(comment.Text, "// apigen:api")
					if needCodegen {
						funcComment = funcComment + strings.ReplaceAll(comment.Text, "// apigen:api", "")
					}
				}
			}
			if !needCodegen {
				fmt.Printf("SKIP func %#v doesnt have apigen mark\n", decl.Name.Name)
				continue
			}

			// Обработка функции
			fmt.Printf("process func %s\n", decl.Name.Name)
			currMethod := &apiMethod{}
			if err := json.Unmarshal([]byte(funcComment), currMethod); err != nil {
				fmt.Printf("err func %s %s\n", decl.Name.Name, err.Error())
			}
			currMethod.Name = decl.Name.Name
			currMethod.ParamsName = decl.Type.Params.List[1].Type.(*ast.Ident).Name
			currMethod.APIName = decl.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
			currMethod.HandlerName = "handler" + currMethod.APIName + currMethod.Name

			apiMethods[currMethod.APIName+currMethod.Name] = currMethod
			apis[currMethod.APIName] = currMethod.APIName

		// ОБРАБАТЫВАЕМ СТРУКТУРЫ
		case *ast.GenDecl:

			for _, spec := range decl.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if !ok {
					fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
					continue
				}

				currStruct, ok := currType.Type.(*ast.StructType)
				if !ok {
					fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
					continue
				}

				apiParams := make([]*apiMethodParams, 0)
				for _, field := range currStruct.Fields.List {

					if field.Tag != nil {
						tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
						tagVal := tag.Get("apivalidator")
						if tagVal == "" || tagVal == "-" {
							continue
						}

						apiParam := &apiMethodParams{}
						apiParam.ParamsName = currType.Name.Name
						apiParam.Name = field.Names[0].Name
						apiParam.Type = field.Type.(*ast.Ident).Name

						tagParams := strings.Split(tagVal, ",")
					PARAMSLOOP:
						for _, tagParam := range tagParams {
							currTag := strings.Split(tagParam, "=")
							switch currTag[0] {
							case "-":
								break PARAMSLOOP
							case "required":
								apiParam.Required = true
							case "paramname":
								apiParam.Paramname = currTag[1]
							case "enum":
								apiParam.Enum = currTag[1]
							case "default":
								apiParam.Default = currTag[1]
							case "min":
								if min, err := strconv.Atoi(currTag[1]); err == nil {
									apiParam.Min = min
									apiParam.HasMin = true
								}
							case "max":
								if max, err := strconv.Atoi(currTag[1]); err == nil {
									apiParam.Max = max
									apiParam.HasMax = true
								}
							}
						}

						if apiParam.Paramname == "" {
							apiParam.Paramname = strings.ToLower(apiParam.Name)
						}

						apiParams = append(apiParams, apiParam)
					}
				}
				apiMethodsParams[currType.Name.Name] = apiParams
			}
		}
	}

	return apis, apiMethods, apiMethodsParams
}

func codegenAPI(out *os.File, apis map[string]string, apiMethods map[string]*apiMethod, apiMethodsParams map[string][]*apiMethodParams) {

	for _, api := range apis {
		handlresCode := ""
		for _, apiMethod := range apiMethods {
			if apiMethod.APIName != api {
				continue
			}
			handlresCode += "\t\tcase \"" + apiMethod.URL + "\": h." + apiMethod.HandlerName + "(w, r)\n"
		}
		serveHTTPTpl.Execute(out, map[string]string{
			"API":      api,
			"Handlers": handlresCode,
		})

		for _, apiMethod := range apiMethods {
			if apiMethod.APIName != api {
				continue
			}

			var currParams bytes.Buffer
			for _, apiMethodParam := range apiMethodsParams[apiMethod.ParamsName] {
				currParams.WriteString("\n\n\t// " + apiMethodParam.Name)
				if apiMethodParam.Type == "int" {
					paramIntTpl.Execute(&currParams, apiMethodParam)
				} else {
					paramStringTpl.Execute(&currParams, apiMethodParam)
				}
				if apiMethodParam.Required {
					paramRequiredTpl.Execute(&currParams, apiMethodParam)
				}
				if apiMethodParam.Default != "" {
					paramDefaultTpl.Execute(&currParams, apiMethodParam)
				}
				if apiMethodParam.Enum != "" {
					paramEnumTpl.Execute(&currParams, map[string]string{
						"Name":      apiMethodParam.Name,
						"Paramname": apiMethodParam.Paramname,
						"Enum":      "|" + apiMethodParam.Enum + "|",
						"EnumErr":   strings.ReplaceAll(apiMethodParam.Enum, "|", ", "),
					})
				}
				if apiMethodParam.HasMin {
					if apiMethodParam.Type == "int" {
						paramMinIntTpl.Execute(&currParams, apiMethodParam)
					} else {
						paramMinStringTpl.Execute(&currParams, apiMethodParam)
					}
				}
				if apiMethodParam.HasMax {
					if apiMethodParam.Type == "int" {
						paramMaxIntTpl.Execute(&currParams, apiMethodParam)
					} else {
						paramMaxStringTpl.Execute(&currParams, apiMethodParam)
					}
				}
			}

			handlerTpl.Execute(out, map[string]interface{}{
				"API":         api,
				"HandlerName": apiMethod.HandlerName,
				"ParamsName":  apiMethod.ParamsName,
				"Params":      currParams.String(),
				"Auth":        apiMethod.Auth,
				"Method":      apiMethod.Method,
				"FuncName":    apiMethod.Name,
			})
		}
	}
	//intTpl.Execute(out, tpl{fieldName})

	fmt.Fprintln(out) // empty line
}

// go build ./handlers_gen/codegen.go
// ./codegen.exe api.go api_handlers.go
