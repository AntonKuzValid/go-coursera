package main

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testUsers = []User{
	{Id: 1, Name: "User1", Age: 1, About: "About1", Gender: "male"},
}

type TestCase struct {
	SearchRequest  *SearchRequest
	SearchResponse *SearchResponse
	AccessToken    string
	IsError        bool
	ErrorMsg       string
}

func MockGetUsers(w http.ResponseWriter, r *http.Request) {

	if token := r.Header.Get("AccessToken"); token != "access" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("query")

	if query == "SELECT BAD_UNPACK_JSON" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if query == "UNPACK_JSON" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if query == "SELECT SERVER_ERROR" {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	if query == "SELECT TIMEOUT" {
		time.Sleep(2 * time.Second)
		return
	}

	if query == "SELECT UNKNOWN" {
		time.Sleep(1 * time.Second)
		return
	}

	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	order_field := r.URL.Query().Get("order_field")
	order_by := r.URL.Query().Get("order_by")

	if limit == "" || offset == "" || query == "" || order_field == "" || order_by == "" {
		err := new(SearchErrorResponse)
		if order_field == "" {
			err.Error = "ErrorBadOrderField"
		} else {
			err.Error = "unknown"
		}
		bytes, _ := json.Marshal(err)
		http.Error(w, string(bytes), http.StatusBadRequest)
		return
	}

	bytes, err := json.Marshal(testUsers)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func TestFindUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(MockGetUsers))
	defer ts.Close()

	testCases := []TestCase{
		{
			SearchRequest: &SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "SELECT 1",
				OrderField: "1",
				OrderBy:    0,
			},
			SearchResponse: &SearchResponse{
				Users:    testUsers,
				NextPage: false,
			},
			AccessToken: "access",
			IsError:     false,
		},
		{
			SearchRequest: &SearchRequest{
				Limit:      0,
				Offset:     0,
				Query:      "SELECT 1",
				OrderField: "1",
				OrderBy:    0,
			},
			SearchResponse: &SearchResponse{
				Users:    testUsers[0 : len(testUsers)-1],
				NextPage: true,
			},
			AccessToken: "access",
			IsError:     false,
		},
		{
			SearchRequest: &SearchRequest{
				Limit:      -1,
				Offset:     0,
				Query:      "SELECT 1",
				OrderField: "1",
				OrderBy:    0,
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "limit must be > 0",
		},
		{
			SearchRequest: &SearchRequest{
				Limit:      30,
				Offset:     -1,
				Query:      "SELECT 1",
				OrderField: "1",
				OrderBy:    0,
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "offset must be > 0",
		},
		{
			SearchRequest: &SearchRequest{
				Query: "SELECT TIMEOUT",
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "timeout for limit=1&offset=0&order_by=0&order_field=&query=SELECT+TIMEOUT",
		},
		{
			SearchRequest: &SearchRequest{
				Query: "SELECT 1",
			},
			SearchResponse: nil,
			AccessToken:    "unauthorized",
			IsError:        true,
			ErrorMsg:       "Bad AccessToken",
		},
		{
			SearchRequest: &SearchRequest{
				Query: "SELECT SERVER_ERROR",
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "SearchServer fatal error",
		},
		{
			SearchRequest: &SearchRequest{
				Query: "SELECT BAD_UNPACK_JSON",
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "cant unpack error json: unexpected end of JSON input",
		},
		{
			SearchRequest: &SearchRequest{
				Query: "UNPACK_JSON",
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "cant unpack result json: unexpected end of JSON input",
		},
		{
			SearchRequest: &SearchRequest{
				Limit:      30,
				Offset:     1,
				Query:      "SELECT 1",
				OrderField: "",
				OrderBy:    100,
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "OrderFeld  invalid",
		},
		{
			SearchRequest: &SearchRequest{
				Limit:      30,
				Offset:     1,
				Query:      "",
				OrderField: "1",
				OrderBy:    100,
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg:       "unknown bad request error: unknown",
		},
		{
			SearchRequest: &SearchRequest{
				Query: "SELECT UNKNOWN",
			},
			SearchResponse: nil,
			AccessToken:    "access",
			IsError:        true,
			ErrorMsg: "unknown error Get " + ts.URL + "?limit=1&offset=0&order_by=0&order_field=&query=SELECT+UNKNOWN: dial tcp " +
				strings.Replace(ts.URL, "http://", "", 1) + ": connect: connection refused",
		},
	}

	//scWithNullUrl := &SearchClient{}
	//_, err := scWithNullUrl.FindUsers(SearchRequest{})
	//if err == nil {
	//	t.Errorf("[%d] error was expected", -1)
	//}
	//
	//s := err.Error()
	//if "error" != s {
	//	t.Errorf("[%d] expected error message  %+v but actual is %+v", -1, "error", err.Error())
	//}

	for caseNum, tc := range testCases {
		if caseNum == 11 {
			ts.Close()
		}
		sc := &SearchClient{URL: ts.URL, AccessToken: tc.AccessToken}
		response, err := sc.FindUsers(*tc.SearchRequest)

		if tc.IsError {
			if err == nil {
				t.Errorf("[%d] error was expected", caseNum)
			}

			s := err.Error()
			if tc.ErrorMsg != s {
				t.Errorf("[%d] expected error message  %+v but actual is %+v", caseNum, tc.ErrorMsg, err.Error())
			}

		} else {
			if err != nil {
				t.Errorf("[%d] unexpected error: %#v", caseNum, err)
			}
			if !cmp.Equal(response, tc.SearchResponse) {
				t.Errorf("[%d] expected  %+v but actual is %+v", caseNum, response, tc.SearchResponse)
			}
		}
	}
}

// код писать тут
