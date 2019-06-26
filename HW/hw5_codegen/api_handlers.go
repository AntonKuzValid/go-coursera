package main

import "encoding/json"
import "net/http"
import "fmt"
import "strconv"

func validateProfileParams(r *http.Request) (*ProfileParams, error) {
	params := &ProfileParams{}
	var err error

	Login = r.FormValue("login")

	if Login == "" {
		return nil, fmt.Errorf("login must me not empty")
	}

	return params, err
}

func validateCreateParams(r *http.Request) (*CreateParams, error) {
	params := &CreateParams{}
	var err error

	Login = r.FormValue("login")

	if Login == "" {
		return nil, fmt.Errorf("login must me not empty")
	}

	if 10 > len([]rune(Login)) {
		return nil, fmt.Errorf("login len must be >= 10")
	}

	Name = r.FormValue("full_name")

	Status = r.FormValue("status")

	if Status == "" {
		Status = "user"
	}

	valid := func(enums []string) bool {
		for _, enum := range enums {
			if Status == enum {
				return true
			}
		}
		return false
	}([]string{"user", "moderator", "admin"})
	if !valid {
		return nil, fmt.Errorf("status must be one of [user, moderator, admin]")
	}

	Age, err = strconv.Atoi(r.Form.Get("age"))
	if err != nil {
		return nil, fmt.Errorf("age must be int")
	}

	if 128 <= Age {
		return nil, fmt.Errorf("age must be <= 128")
	}

	if 0 > Age {
		return nil, fmt.Errorf("age must be >= 0")
	}

	return params, err
}

func validateOtherCreateParams(r *http.Request) (*OtherCreateParams, error) {
	params := &OtherCreateParams{}
	var err error

	Username = r.FormValue("username")

	if Username == "" {
		return nil, fmt.Errorf("username must me not empty")
	}

	if 3 > len([]rune(Username)) {
		return nil, fmt.Errorf("username len must be >= 3")
	}

	Name = r.FormValue("account_name")

	Class = r.FormValue("class")

	if Class == "" {
		Class = "warrior"
	}

	valid := func(enums []string) bool {
		for _, enum := range enums {
			if Class == enum {
				return true
			}
		}
		return false
	}([]string{"warrior", "sorcerer", "rouge"})
	if !valid {
		return nil, fmt.Errorf("class must be one of [warrior, sorcerer, rouge]")
	}

	Level, err = strconv.Atoi(r.Form.Get("level"))
	if err != nil {
		return nil, fmt.Errorf("level must be int")
	}

	if 50 <= Level {
		return nil, fmt.Errorf("level must be <= 50")
	}

	if 1 > Level {
		return nil, fmt.Errorf("level must be >= 1")
	}

	return params, err
}

func (srv *MyApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {

	case "/user/profile":

		params, err := validateProfileParams(r)
		if err != nil {
			errorMsg, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(rw, string(errorMsg), http.StatusBadRequest)
			return
		}
		rval, err := Profile(r.Context(), *params)
		if err != nil {
			if apiError, ok := err.(ApiError); ok {
				errorMsg, _ := json.Marshal(map[string]interface{}{"error": Error()})
				http.Error(rw, string(errorMsg), HTTPStatus)
			} else {
				errorMsg, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
				http.Error(rw, string(errorMsg), http.StatusInternalServerError)
			}
			return
		}
		data, err := json.Marshal(map[string]interface{}{"error": "", "response": rval})
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		rw.Write(data)

	case "/user/create":

		if r.Method == "POST" {

			if r.Header.Get("X-Auth") != "100500" {
				errorMsg, _ := json.Marshal(map[string]interface{}{"error": "unauthorized"})
				http.Error(rw, string(errorMsg), http.StatusForbidden)
				return
			}

			params, err := validateCreateParams(r)
			if err != nil {
				errorMsg, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
				http.Error(rw, string(errorMsg), http.StatusBadRequest)
				return
			}
			rval, err := Create(r.Context(), *params)
			if err != nil {
				if apiError, ok := err.(ApiError); ok {
					errorMsg, _ := json.Marshal(map[string]interface{}{"error": Error()})
					http.Error(rw, string(errorMsg), HTTPStatus)
				} else {
					errorMsg, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
					http.Error(rw, string(errorMsg), http.StatusInternalServerError)
				}
				return
			}
			data, err := json.Marshal(map[string]interface{}{"error": "", "response": rval})
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			rw.Write(data)

		} else {
			errorMsg, _ := json.Marshal(map[string]interface{}{"error": "bad method"})
			http.Error(rw, string(errorMsg), http.StatusNotAcceptable)
		}

	default:
		errorMsg, _ := json.Marshal(map[string]interface{}{"error": "unknown method"})
		http.Error(rw, string(errorMsg), http.StatusNotFound)
	}
}

func (srv *OtherApi) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {

	case "/user/create":

		if r.Method == "POST" {

			if r.Header.Get("X-Auth") != "100500" {
				errorMsg, _ := json.Marshal(map[string]interface{}{"error": "unauthorized"})
				http.Error(rw, string(errorMsg), http.StatusForbidden)
				return
			}

			params, err := validateOtherCreateParams(r)
			if err != nil {
				errorMsg, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
				http.Error(rw, string(errorMsg), http.StatusBadRequest)
				return
			}
			rval, err := Create(r.Context(), *params)
			if err != nil {
				if apiError, ok := err.(ApiError); ok {
					errorMsg, _ := json.Marshal(map[string]interface{}{"error": Error()})
					http.Error(rw, string(errorMsg), HTTPStatus)
				} else {
					errorMsg, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
					http.Error(rw, string(errorMsg), http.StatusInternalServerError)
				}
				return
			}
			data, err := json.Marshal(map[string]interface{}{"error": "", "response": rval})
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			rw.Write(data)

		} else {
			errorMsg, _ := json.Marshal(map[string]interface{}{"error": "bad method"})
			http.Error(rw, string(errorMsg), http.StatusNotAcceptable)
		}

	default:
		errorMsg, _ := json.Marshal(map[string]interface{}{"error": "unknown method"})
		http.Error(rw, string(errorMsg), http.StatusNotFound)
	}
}
