package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClient(t *testing.T) {
	cases := []TestCase{
		TestCase{
			ID: "42",
			Result: &CheckoutResult{
				Status:  200,
				Balance: 100500,
				Err:     "",
			},
			IsError: false,
		},
		TestCase{
			ID: "100500",
			Result: &CheckoutResult{
				Status:  400,
				Balance: 0,
				Err:     "bad_balance",
			},
			IsError: false,
		},
		TestCase{
			ID:      "__broken_json",
			Result:  nil,
			IsError: true,
		},
		TestCase{
			ID:      "__internal_error",
			Result:  nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(FindUsers))

	for caseNum, item := range cases {
		c := &Cart{
			PaymentApiURL: ts.URL,
		}
		result, err := c.Checkout(item.ID)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}
