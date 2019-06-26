package main

import (
	"github.com/prometheus/common/log"
	"os"
	"testing"
)

func TestHandlerTemplate(t *testing.T) {
	if err := handlerTpl.Execute(os.Stdout, tmpl{
		ApiName: "MyApi",
		Apis: []api{
			{
				Url:       "/user/profile",
				Auth:      false,
				Method:    "",
				FuncName:  "Profile",
				ParamName: "ProfileParams",
			},
			{
				Url:       "/user/create",
				Auth:      true,
				Method:    "POST",
				FuncName:  "Create",
				ParamName: "CreateParams",
			},
		},
	}); err != nil {
		log.Fatal(err)
	}
}

func TestValidateTemplate(t *testing.T) {
	if err := validateTpl.Execute(os.Stdout, validateTmpl{
		ParamName: "",
		Fields: []fieldTpl{{
			Name:         "name",
			ParamName:    "new_name",
			Default:      "0",
			Required:     true,
			IsEnum:       true,
			Enums:        "1,2",
			IsDefault:    true,
			DefaultValue: "default",
		}},
	}); err != nil {
		log.Fatal(err)
	}
}
