package main

import (
	"time"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"encoding/xml"
	"net/http/httptest"
	"reflect"
	"testing"
	"io"
	"fmt"
	"strings"
	"strconv"
)

type TestCase struct {
	AccessToken string
	SearchRequest *SearchRequest
	Result  *SearchResponse
	IsError bool
}

type FindUserResult struct {
	NextPage  bool
	Users []User
}

type UserXML struct {
	ID 						int			`xml:"id"`
	GUID 					string	`xml:"guid"`
	IsActive 			bool		`xml:"isActive"`
	Balance 			string	`xml:"balance"`
	Picture 			string	`xml:"picture"`
	Age 					int			`xml:"age"`
	EyeColor 			string	`xml:"eyeColor"`
	FirstName 		string	`xml:"first_name"`
	LastName 			string	`xml:"last_name"`
	Gender 				string	`xml:"gender"`
	Email 				string	`xml:"email"`
	Phone 				string	`xml:"phone"`
	Addres 				string	`xml:"addres"`
	About 				string	`xml:"about"`
	Registered 		string	`xml:"registered"`
	FavoriteFruit	string	`xml:"favoriteFruit"`
}

type UsersXML struct {
	Users	[]UserXML `xml:"row"`
}

func UsersDecoder() ([]UserXML) {
	xmlData, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		panic(err)
	}
	v := new(UsersXML)
	err = xml.Unmarshal(xmlData, &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return nil
	}
	return v.Users
}

func SearchServer(w http.ResponseWriter, r *http.Request){
	// Данные для работы лежаит в файле `dataset.xml`
	// Параметр `query` ищет по полям `Name` и `About`
	// Параметр `order_field` работает по полям `Id`, `Age`, `Name`, если пустой - то возвращаем по `Name`, если что-то другое - SearchServer ругается ошибкой. `Name` - это first_name + last_name из xml. 
	// Если `query` пустой, то делаем только сортировку, т.е. возвращаем все записи
	// Как работать с XML смотрите в `xml/*`

	accessToken := r.Header.Get("AccessToken")
	switch accessToken {
	case "internal_error": 
		w.WriteHeader(http.StatusInternalServerError)
		return
	case "unknown_error":
		//w.WriteHeader(http.Status)
		io.WriteString(w, "unknown error")
		return
	case "bad_request":
		result, err := json.Marshal(SearchErrorResponse{
			Error: "ErrorBadOrderField",
		})
		w.WriteHeader(http.StatusBadRequest)
		if err == nil {
			io.WriteString(w, string(result))
		}
		return
	case "bad_request_unknown":
		result, err := json.Marshal(SearchErrorResponse{
			Error: "bad_request",
		})
		w.WriteHeader(http.StatusBadRequest)
		if err == nil {
			io.WriteString(w, string(result))
		}
		return
	case "bad_request_cant_unpack":
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "cant_unpack_json")
		return
	case "cant_unpack_json":
		//w.WriteHeader(http.Status)
		io.WriteString(w, "cant_unpack_json")
		return
	case "unauthorized":
		w.WriteHeader(http.StatusUnauthorized)
		return
	case "timeout":
		to := time.After(2*time.Second)
		<- to
	}

	limit, err :=  strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//offset :=  r.URL.Query().Get("offset")
	query :=  r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	//orderBy := r.URL.Query().Get("order_by")


	// Проверяем корректность переданных параметров
	if (orderField != "" && orderField != "Id" && orderField != "Age"  && orderField != "Name") {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Парсим исходные данные 
	// TODO Кешировать
	data := UsersDecoder()

	resultSearch := []User{}
	for _, user := range data {
		if limit <= 0 {
			break
		}
		name := user.FirstName + " " + user.LastName
		// Фильтр по параметру query
		if (query != "" && !strings.Contains(name, query) && !strings.Contains(user.About, query)) {
			continue
		}

		limit--
		// Добавляем в результат
		resultSearch = append(resultSearch, User{
			Id: user.ID,
			Name: name,
			Age: user.Age,
			About: user.About,
			Gender: user.Gender,
		})
	}

	result, err := json.Marshal(resultSearch)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, string(result))
	}	
}

func TestServerErrors(t *testing.T) {
	cases := []TestCase{
		TestCase{
			AccessToken: "timeout",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "unknown_error",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "bad_request",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "bad_request_unknown",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "bad_request_cant_unpack",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "unauthorized",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "internal_error",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "cant_unpack_json",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
	}

	TestingSearchServer(t, cases)
}

func TestSearch(t *testing.T) {
	cases := []TestCase{
		TestCase{
			AccessToken: "single",
			SearchRequest: &SearchRequest{
				Query: "Boyd",
				Limit: 10,
				Offset: 0,
			},
			Result: &SearchResponse{
				Users: []User{
					User{
						Id: 0, Name: "Boyd Wolf", Age: 22, Gender: "male",
						About: "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
					},
				},
				NextPage: false,
			},
			IsError: false,
		},
		TestCase{
			AccessToken: "multi",
			SearchRequest: &SearchRequest{
				Query: "Nulla",
				Limit: 1,
				Offset: 0,
			},
			Result: &SearchResponse{
				Users: []User{
					User{
						Id: 0, Name: "Boyd Wolf", Age: 22, Gender: "male",
						About: "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
					},
				},
				NextPage:  true,
			},
			IsError: false,
		},
		TestCase{
			AccessToken: "limit_error",
			SearchRequest: &SearchRequest{
				Limit: -10,
				Offset: 0,
			},
			Result: nil,
			IsError: true,
		},
		TestCase{
			AccessToken: "offset_error",
			SearchRequest: &SearchRequest{
				Limit: 50,
				Offset: -10,
			},
			Result: nil,
			IsError: true,
		},
	}

	TestingSearchServer(t, cases)
}

func TestUnknownError(t *testing.T) {
	cases := []TestCase{
		TestCase{
			AccessToken: "_",
			SearchRequest: &SearchRequest{},
			Result: nil,
			IsError: true,
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken:  item.AccessToken,
			URL: "",//ts.URL,
		}
		result, err := c.FindUsers(*item.SearchRequest)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		} else if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		} else if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		} else {
			fmt.Printf("[%d] OK - %#v (%#v)\n", caseNum, item.AccessToken, fmt.Sprint(err))
		}

	}
	ts.Close()
}

func TestingSearchServer(t *testing.T, cases []TestCase) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken:  item.AccessToken,
			URL: ts.URL,
		}
		result, err := c.FindUsers(*item.SearchRequest)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		} else if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		} else if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		} else {
			fmt.Printf("[%d] OK - %#v (%#v)\n", caseNum, item.AccessToken, fmt.Sprint(err))
		}

	}
	ts.Close()
}

// Pfib,bcm1=Pfib,bcm
