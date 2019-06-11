package main

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/http/httptest"
	"testing"
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
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	query := r.URL.Query().Get("query")
	order_field := r.URL.Query().Get("order_field")
	order_by := r.URL.Query().Get("order_by")

	if limit == "" || offset == "" || query == "" || order_field == "" || order_by == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
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
	}

	for caseNum, tc := range testCases {
		sc := &SearchClient{URL: ts.URL, AccessToken: tc.AccessToken}
		response, err := sc.FindUsers(*tc.SearchRequest)
		if tc.IsError {
			if err == nil {
				t.Errorf("[%d] error was expected", caseNum)
			}

			if tc.ErrorMsg != err.Error() {
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
