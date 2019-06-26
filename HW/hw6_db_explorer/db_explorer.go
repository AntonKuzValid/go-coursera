package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	mux := http.NewServeMux()

	tables, _ := getTables(db)
	tableNames := make([]string, len(tables))
	for i := range tables {
		tableNames[i] = tables[i].Name
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, `{"error": "unknown Table"}`, http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			bytes, _ := json.Marshal(makeResponse(map[string]interface{}{"tables": tableNames}))
			w.Write(bytes)
		default:
			http.Error(w, "method is not allowed", http.StatusMethodNotAllowed)
		}
	})
	for _, table := range tables {
		cols, err := getColumns(db, table)
		if err != nil {
			return nil, err
		}
		table.Columns = cols
		table.ColumnsMap = make(map[string]*Column, len(cols))
		for _, c := range cols {
			table.ColumnsMap[c.Field] = c
		}

		validMap := make(map[string]string, len(cols))
		fields := make([]string, 0, len(cols))
		placeHolders := make([]string, 0, len(cols))
		keyNumber := make(map[string]int, len(cols))
		for ind := range cols {
			switch {
			case strings.Contains(cols[ind].Type, "varchar") || strings.Contains(cols[ind].Type, "text"):
				validMap[cols[ind].Field] = "string"
			case strings.Contains(cols[ind].Type, "int"):
				validMap[cols[ind].Field] = "int"
			case strings.Contains(cols[ind].Type, "float"):
				validMap[cols[ind].Field] = "int"
			}
			if cols[ind].Key != "PRI" {
				fields = append(fields, cols[ind].Field)
				placeHolders = append(placeHolders, "?")
				keyNumber[cols[ind].Field] = len(fields) - 1
			}
		}
		table.KeyNumber = keyNumber
		table.ValidMap = validMap

		insertStm := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table.Name,
			strings.Join(fields, ","), strings.Join(placeHolders, ","))

		th := TableHandler{DB: db, Table: table, InsertStm: insertStm}
		mux.HandleFunc("/"+table.Name, th.HandleRows)
		mux.HandleFunc("/"+table.Name+"/", th.HandleRowById)
	}
	return mux, nil
}

type TableHandler struct {
	DB        *sql.DB
	Table     *Table
	InsertStm string
}

type Table struct {
	Name       string
	Columns    []*Column
	ColumnsMap map[string]*Column
	KeyNumber  map[string]int
	ValidMap   map[string]string
}

type Column struct {
	Field      string
	Type       string
	Collations sql.NullString
	Null       string
	Key        string
	Default    sql.NullString
	Extra      string
	Privileges string
	Comment    string
}

func (th *TableHandler) HandleRows(w http.ResponseWriter, r *http.Request) {

	db := th.DB
	switch r.Method {

	case http.MethodGet:

		var (
			limit  int
			offset int
			err    error
		)
		if limit, err = strconv.Atoi(r.FormValue("limit")); err != nil {
			limit = 5
		}
		if offset, err = strconv.Atoi(r.FormValue("offset")); err != nil {
			offset = 0
		}

		rows, err := db.Query(fmt.Sprintf("select * from %s limit ? offset ?", th.Table.Name), limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		result := make([]map[string]interface{}, 0, 8)

		cols, _ := rows.Columns()
		for rows.Next() {
			row := make([]interface{}, len(cols))
			for idx := range cols {
				row[idx] = new(MetalScanner)
			}

			err := rows.Scan(row...)
			if err != nil {
				fmt.Println(err)
			}

			rowMap := make(map[string]interface{}, 8)

			for idx, column := range cols {
				var scanner = row[idx].(*MetalScanner)
				rowMap[column] = scanner.value
			}
			result = append(result, rowMap)
		}

		bytes, err := json.Marshal(makeResponse(map[string]interface{}{"records": result}))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(bytes)

	default:
		http.Error(w, "method is not allowed", http.StatusMethodNotAllowed)
	}

}

func (th *TableHandler) HandleRowById(w http.ResponseWriter, r *http.Request) {
	db := th.DB
	switch r.Method {

	case http.MethodGet:
		cols := th.Table.Columns
		uris := strings.Split(r.RequestURI, "/")
		if len(uris) > 2 {
			id := uris[2]
			var keyName string
			for _, col := range th.Table.Columns {
				if col.Key == "PRI" {
					keyName = col.Field
					break
				}
			}
			row := db.QueryRow(fmt.Sprintf("select * from %s where %s=?", th.Table.Name, keyName), id)
			rowResult := make([]interface{}, len(cols))
			for idx := range cols {
				rowResult[idx] = new(MetalScanner)
			}

			err := row.Scan(rowResult...)

			if err != nil {
				if err != sql.ErrNoRows {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				} else {
					http.Error(w, `{"error": "record not found"}`, http.StatusNotFound)
				}
				return
			}
			rowMap := make(map[string]interface{}, 8)
			for idx, column := range cols {
				var scanner = rowResult[idx].(*MetalScanner)
				rowMap[column.Field] = scanner.value
			}
			bytes, err := json.Marshal(makeResponse(map[string]interface{}{"record": rowMap}))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(bytes)

		}

	case http.MethodPut:
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		defer r.Body.Close()
		data := make(map[string]interface{})
		err = json.Unmarshal(bytes, &data)
		values := make([]interface{}, len(th.Table.KeyNumber))

		for _, v := range th.Table.Columns {
			_, ok := data[v.Field]
			if v.Null == "NO" && !ok && v.Key != "PRI" {
				data[v.Field] = ""
			}
		}
		for k, v := range data {
			switch v.(type) {
			case string:
				if t, ok := th.Table.ValidMap[k]; ok {
					if t != "string" {
						http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, k), http.StatusBadRequest)
						return
					}
				}
			case float64:
				if t, ok := th.Table.ValidMap[k]; ok {
					if t != "int" {
						http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, k), http.StatusBadRequest)
						return
					}
				}
			}
			if numb, ok := th.Table.KeyNumber[k]; ok {
				values[numb] = v
			}
		}

		res, err := db.Exec(th.InsertStm, values...)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var keyName string
		for _, col := range th.Table.Columns {
			if col.Key == "PRI" {
				keyName = col.Field
				break
			}
		}
		lastInd, _ := res.LastInsertId()
		json, _ := json.Marshal(makeResponse(map[string]interface{}{keyName: lastInd}))
		w.Write(json)

	case http.MethodPost:
		uris := strings.Split(r.RequestURI, "/")
		if len(uris) > 2 {
			id := uris[2]
			var keyName string
			for _, col := range th.Table.Columns {
				if col.Key == "PRI" {
					keyName = col.Field
					break
				}
			}
			bytes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			defer r.Body.Close()
			data := make(map[string]interface{})
			err = json.Unmarshal(bytes, &data)

			if _, ok := data[keyName]; ok {
				http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, keyName), http.StatusBadRequest)
				return
			}

			placeHolders := make([]string, 0, len(data))
			values := make([]interface{}, 0, len(data)+1)

			for k, v := range data {
				switch v.(type) {
				case string:
					if t, ok := th.Table.ValidMap[k]; ok {
						if t != "string" {
							http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, k), http.StatusBadRequest)
							return
						}
					}
				case float64:
					if t, ok := th.Table.ValidMap[k]; ok {
						if t != "int" {
							http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, k), http.StatusBadRequest)
							return
						}
					}
				case nil:
					if th.Table.ColumnsMap[k].Null == "NO" {
						http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, k), http.StatusBadRequest)
						return
					}
				}
				placeHolders = append(placeHolders, k+"=?")
				values = append(values, v)
			}
			values = append(values, id)
			sqlUpdate := fmt.Sprintf("update %s set %s where %s=?", th.Table.Name, strings.Join(placeHolders, ","), keyName)
			res, err := db.Exec(sqlUpdate, values...)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			aff, _ := res.RowsAffected()
			json, _ := json.Marshal(makeResponse(map[string]interface{}{"updated": aff}))

			w.Write(json)
		}

	case http.MethodDelete:
		uris := strings.Split(r.RequestURI, "/")
		if len(uris) > 2 {
			id := uris[2]
			var keyName string
			for _, col := range th.Table.Columns {
				if col.Key == "PRI" {
					keyName = col.Field
					break
				}
			}
			res, err := db.Exec(fmt.Sprintf("delete from %s where %s=?", th.Table.Name, keyName), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			numb, _ := res.RowsAffected()
			json, _ := json.Marshal(makeResponse(map[string]interface{}{"deleted": numb}))
			w.Write(json)
		}

	default:
		http.Error(w, "method is not allowed", http.StatusMethodNotAllowed)
	}
}

func getTables(db *sql.DB) ([]*Table, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]*Table, 0, 8)
	for rows.Next() {
		table := Table{}
		if err := rows.Scan(&table.Name); err != nil {
			return nil, err
		}
		tables = append(tables, &table)
	}
	return tables, nil
}

func getColumns(db *sql.DB, table *Table) ([]*Column, error) {
	rows, err := db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM %s", table.Name))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make([]*Column, 0, 8)
	for rows.Next() {
		c := Column{}
		if err := rows.Scan(&c.Field, &c.Type, &c.Collations, &c.Null, &c.Key, &c.Default, &c.Extra, &c.Privileges, &c.Comment); err != nil {
			return nil, err
		}
		columns = append(columns, &c)
	}
	return columns, nil

}

func makeResponse(res interface{}) map[string]interface{} {
	return map[string]interface{}{"response": res}
}

type MetalScanner struct {
	valid bool
	value interface{}
}

func (scanner *MetalScanner) Scan(src interface{}) error {
	switch src.(type) {
	case []uint8:
		scanner.valid = true
		scanner.value = string(src.([]uint8))
	case int64:
		scanner.valid = true
		scanner.value = src.(int64)
	case int32:
		scanner.valid = true
		scanner.value = src.(int32)
	}
	return nil
}
