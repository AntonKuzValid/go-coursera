package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

// код писать тут

const (
	API_PREFIX       = "// apigen:api "
	VALIDATOR_PREFIX = "apivalidator"
)

var (
	REQUIRED   = regexp.MustCompile("required")
	MIN        = regexp.MustCompile("min=(.+)")
	MAX        = regexp.MustCompile("max=(.+)")
	PARAM_NAME = regexp.MustCompile("paramname=(.+)")
	DEFAULT    = regexp.MustCompile("default=(.+)")
	ENUM       = regexp.MustCompile("enum=(.+)")
)

type api struct {
	Url       string `json:"url"`
	Auth      bool   `json:"auth"`
	Method    string `json:"method,omitempty"`
	FuncName  string
	ParamName string
}

type tmpl struct {
	ApiName string
	Apis    []api
}

var handlerTpl = template.Must(template.New("handlerTpl").Parse(`
func (srv *{{.ApiName}}) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
switch r.URL.Path {
{{range $i, $a := .Apis}}
case "{{$a.Url}}":
	{{if $a.Method}}
	if r.Method == "{{$a.Method}}" {
	{{end}}
		{{if $a.Auth}}
		if r.Header.Get("X-Auth") != "100500" {
			errorMsg, _ := json.Marshal(map[string]interface{}{"error": "unauthorized"})
			http.Error(rw, string(errorMsg), http.StatusForbidden)
			return
		}
		{{end}}
		params, err := validate{{$a.ParamName}}(r)
		if err != nil {
			errorMsg, _ := json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(rw, string(errorMsg), http.StatusBadRequest)
			return
		}
		rval, err := srv.{{$a.FuncName}}(r.Context(), *params)
		if err != nil {
			if apiError, ok := err.(ApiError); ok {
				errorMsg, _ := json.Marshal(map[string]interface{}{"error": apiError.Error()})
				http.Error(rw, string(errorMsg), apiError.HTTPStatus)
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
	{{if $a.Method}}
	} else {
		errorMsg, _ := json.Marshal(map[string]interface{}{"error": "bad method"})
		http.Error(rw, string(errorMsg), http.StatusNotAcceptable)
	}
	{{end}}
{{end}}
default:
	errorMsg, _ := json.Marshal(map[string]interface{}{"error": "unknown method"})
		http.Error(rw, string(errorMsg), http.StatusNotFound)
}
}
`))

type fieldTpl struct {
	Name         string
	ParamName    string
	Default      string
	Required     bool
	IsEnum       bool
	Enums        string
	EnumType     string
	IsDefault    bool
	DefaultValue string
	IsMax        bool
	Max          string
	IsMin        bool
	Min          string
	EnumError    string
}

type validateTmpl struct {
	ParamName string
	Fields    []fieldTpl
}

var validateTpl = template.Must(template.New("validateTpl").Parse(`
func validate{{.ParamName}}(r *http.Request) (*{{.ParamName}}, error) {
params := &{{.ParamName}}{}
var err error
{{range $i, $f := .Fields}}
	params.{{$f.Name}}{{if eq $f.Default "0"}}, err = strconv.Atoi(r.Form.Get("{{$f.ParamName}}"))
	if err!=nil {
		return nil, fmt.Errorf("{{$f.ParamName}} must be int")
	}{{else}} = r.FormValue("{{$f.ParamName}}"){{end}}
	{{if $f.IsDefault}}
	if params.{{$f.Name}} == {{$f.Default}} {
		params.{{$f.Name}} = {{$f.DefaultValue}}
	}
	{{end}}
	{{if $f.Required}}
	if params.{{$f.Name}} == {{$f.Default}} {
		return nil, fmt.Errorf("{{$f.ParamName}} must me not empty")
	}
	{{end}}
	{{if $f.IsEnum}}
	valid := func(enums []{{$f.EnumType}}) bool{
		for _, enum := range enums{
			if params.{{$f.Name}} == enum {
				return true
			}
		}
		return false
	}([]{{$f.EnumType}}{ {{$f.Enums}} })
	if !valid {
		return nil, fmt.Errorf("{{$f.EnumError}}")
	}
	{{end}}
	{{if $f.IsMax}}
	if {{$f.Max}} <= {{if eq $f.Default "0"}} params.{{$f.Name}} {{else}} len([]rune(params.{{$f.Name}})) {{end}}{
		return nil, fmt.Errorf("{{$f.ParamName}} {{if eq $f.Default "0"}}{{else}}len {{end}}must be <= {{$f.Max}}")
	}
	{{end}}
	{{if $f.IsMin}}
	if {{$f.Min}} > {{if eq $f.Default "0"}} params.{{$f.Name}} {{else}} len([]rune(params.{{$f.Name}})) {{end}}{
		return nil, fmt.Errorf("{{$f.ParamName}} {{if eq $f.Default "0"}}{{else}}len {{end}}must be >= {{$f.Min}}")
	}
	{{end}}
{{end}}
return params, err
}
`))

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	out, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(out, "package "+node.Name.Name)
	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, `import "encoding/json"`)
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "fmt"`)
	fmt.Fprintln(out, `import "strconv"`)
	fmt.Fprintln(out) // empty line

	tmpls := make(map[string]*tmpl, 4)
	validateTmpls := make(map[string]*validateTmpl, 4)

NODE_LOOP:
	for _, f := range node.Decls {

		switch f.(type) {
		case *ast.GenDecl:
			g := f.(*ast.GenDecl)
			//SPECS_LOOP:
			for _, spec := range g.Specs {
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

				typeName := currType.Name.Name
				fmt.Printf("process struct %s\n", typeName)

				methodName := "validate" + typeName

				tmpl := &validateTmpl{
					ParamName: typeName,
					Fields:    make([]fieldTpl, 0, len(currStruct.Fields.List)),
				}

				//FIELD_LOOP:
				for _, field := range currStruct.Fields.List {

					if field.Tag == nil {
						continue NODE_LOOP
					}

					tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
					tagValue, ok := tag.Lookup(VALIDATOR_PREFIX)
					if !ok {
						continue NODE_LOOP
					}
					fieldtmpl := fieldTpl{}

					fieldtmpl.Name = field.Names[0].Name
					fieldtmpl.ParamName = strings.ToLower(fieldtmpl.Name)
					fileType := field.Type.(*ast.Ident).Name

					switch fileType {
					case "int":
						fieldtmpl.Default = "0"
					case "string":
						fieldtmpl.Default = `""`
					default:
						log.Fatalln("unsupported", fileType)
					}

					tagValues := strings.Split(tagValue, ",")

					for _, value := range tagValues {
						if paramArray := PARAM_NAME.FindAllStringSubmatch(value, 1); len(paramArray) > 0 {
							fieldtmpl.ParamName = paramArray[0][1]
						}

						if defaultArray := DEFAULT.FindAllStringSubmatch(value, 1); len(defaultArray) > 0 {
							fieldtmpl.IsDefault = true
							defaultValue := defaultArray[0][1]
							if fileType == "string" {
								defaultValue = fmt.Sprintf(`"%s"`, defaultValue)
							}
							fieldtmpl.DefaultValue = defaultValue
						}

						//check if required
						if REQUIRED.MatchString(value) {
							fieldtmpl.Required = true
						}

						//check is in enum
						if enumArray := ENUM.FindAllStringSubmatch(value, 1); len(enumArray) > 0 {
							enums := strings.Split(enumArray[0][1], "|")
							fieldtmpl.EnumError = fmt.Sprintf(`%s must be one of [%s]`,
								fieldtmpl.ParamName, strings.Join(enums, ", "))
							if fileType == "string" {
								for i := range enums {
									enums[i] = fmt.Sprintf(`"%s"`, enums[i])
								}
							}
							fieldtmpl.IsEnum = true
							fieldtmpl.EnumType = fileType
							fieldtmpl.Enums = strings.Join(enums, ",")
						}

						if maxArray := MAX.FindAllStringSubmatch(value, 1); len(maxArray) > 0 {
							fieldtmpl.IsMax = true
							fieldtmpl.Max = maxArray[0][1]
						}

						if minArray := MIN.FindAllStringSubmatch(value, 1); len(minArray) > 0 {
							fieldtmpl.IsMin = true
							fieldtmpl.Min = minArray[0][1]
						}
					}
					tmpl.Fields = append(tmpl.Fields, fieldtmpl)
					fmt.Printf("\tgenerating code for field %s.%s\n", typeName, fieldtmpl.Name)
				}
				if len(tmpl.Fields) > 0 {
					validateTmpls[methodName] = tmpl
				}
			}

		case *ast.FuncDecl:
			g := f.(*ast.FuncDecl)
			rName := getReciever(g)
			if rName != "" {
				capi, err := getApi(g)
				if err != nil {
					continue
				}
				if tmp, ok := tmpls[rName]; ok {
					tmp.Apis = append(tmp.Apis, *capi)
				} else {
					tmpls[rName] = &tmpl{
						ApiName: rName,
						Apis:    []api{*capi},
					}
				}
			}

		default:
			fmt.Printf("SKIP %T is not *ast.GenDecl or *ast.FuncDecl\n", f)
			continue
		}
	}

	for _, print := range validateTmpls {
		validateTpl.Execute(out, print)
	}

	for _, print := range tmpls {
		handlerTpl.Execute(out, print)
	}
}

func getReciever(fd *ast.FuncDecl) string {
	if fd.Recv != nil {
		for _, r := range fd.Recv.List {
			if expr, ok := r.Type.(*ast.StarExpr); ok {
				if ident, ok := expr.X.(*ast.Ident); ok {
					return ident.Name
				}
			}
		}
	}
	return ""
}

func getApi(fd *ast.FuncDecl) (*api, error) {
	if fd.Doc != nil {
		for _, l := range fd.Doc.List {
			if len(l.Text) > 0 && strings.HasPrefix(l.Text, API_PREFIX) {
				api := api{}
				data := strings.Replace(l.Text, API_PREFIX, "", -1)
				if err := json.Unmarshal([]byte(data), &api); err != nil {
					return nil, err
				}
				api.FuncName = fd.Name.Name
				for _, param := range fd.Type.Params.List {
					if ident, ok := param.Type.(*ast.Ident); ok {
						api.ParamName = ident.Name
					}
				}
				return &api, nil
			}
		}
	}
	return nil, fmt.Errorf("no comment with %s", API_PREFIX)
}
