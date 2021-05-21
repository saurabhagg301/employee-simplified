/*
POST
------
curl -sX POST http://localhost:8085/employee -d '{"name":"Bob", "age":30}' | jq
curl -sX POST http://localhost:8085/employee -d '{"name":"Sara", "age":34}' | jq
curl -sX POST http://localhost:8085/employee -d '{"name":"Mike", "age":36}' | jq

PUT
----
curl -sX PUT http://localhost:8085/employee/1 -d '{"name":"Bob2", "age":32}' | jq
curl -sX PUT http://localhost:8085/employee/1 -d '{"age":34}' | jq

PATCH
--------
curl -sX PATCH http://localhost:8085/employee/2 -d '{"age":70}' | jq


GET
---
curl -sX GET http://localhost:8085/employees | jq
curl -sX GET http://localhost:8085/employee/1 | jq
curl -sX GET http://localhost:8085/employee/Bob | jq

DELETE
----------
curl -sX DELETE http://localhost:8085/employee/Bob | jq
curl -sX DELETE http://localhost:8085/employee/3 | jq
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type employee struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// global variables
var (
	employeeDB = []employee{}
	empIDSeq   = 0
	host       = "127.0.0.1"
	port       = 8085
)

func main() {
	r := mux.NewRouter()
	srvr := http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      r, // *** To attach mux router to server
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	r.HandleFunc("/employee", AddEmployee).Methods("POST")
	r.HandleFunc("/employees", GetEmployees).Methods("GET")
	r.HandleFunc("/employee/{nameORId}", GetEmployee).Methods("GET") // *** Note that query params should be enclosed within curly braces
	r.HandleFunc("/employee/{id}", UpdateEmployee).Methods("PUT")
	r.HandleFunc("/employee/{id}", PartialUpdateEmployee).Methods("PATCH")
	r.HandleFunc("/employee/{nameORId}", DeleteEmployee).Methods("DELETE")

	log.Fatal(srvr.ListenAndServe())

}

func webJSONResponse(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload) // To marshal the payload into a json
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8") // *** To set the content type
	w.Write(response)
}

// GetEmployees to get all employees
func GetEmployees(w http.ResponseWriter, r *http.Request) {
	webJSONResponse(w, http.StatusOK, map[string]interface{}{"employees": employeeDB})
}

// AddEmployee to add an employee
func AddEmployee(w http.ResponseWriter, r *http.Request) {
	e := employee{}
	errDecode := json.NewDecoder(r.Body).Decode(&e) // ***For decoding request payload
	if errDecode != nil {
		webJSONResponse(w, 400, map[string]interface{}{"error": "Failed to decode request payload"})
		return
	}
	empIDSeq++
	e.Id = empIDSeq
	employeeDB = append(employeeDB, e)
	msg := fmt.Sprintf("Employee with id %d created successfully", empIDSeq)
	webJSONResponse(w, 201, map[string]interface{}{"created": msg})
}

func GetEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nameORId := vars["nameORId"]
	id, errDecodeID := strconv.ParseInt(nameORId, 10, 64)
	var e employee
	var statusCode int
	var err error
	if id > 0 && errDecodeID == nil {
		// input query param is a id
		// call getEmployeeByID
		e, statusCode, err = getEmployeeByID(id)
		if err != nil {
			webJSONResponse(w, statusCode, map[string]interface{}{"error": err.Error()})
			return
		}
	} else {
		// input query param is a name
		// call deleteEmployeeByName
		e, statusCode, err = getEmployeeByName(nameORId)
		if err != nil {
			webJSONResponse(w, statusCode, map[string]interface{}{"error": err.Error()})
			return
		}
	}

	// return
	webJSONResponse(w, http.StatusOK, map[string]interface{}{"employee": e})
}

func getEmployeeByID(id int64) (employee, int, error) {
	flagFound := false
	var res employee
	for _, v := range employeeDB {
		if v.Id == int(id) {
			flagFound = true
			res = v
			break
		}
	}
	if !flagFound {
		return res, http.StatusNotFound, errors.New(fmt.Sprintf("No record exists for id %d", id))
	}

	return res, http.StatusOK, nil
}

func getEmployeeByName(name string) (employee, int, error) {
	flagFound := false
	var res employee
	for _, v := range employeeDB {
		if v.Name == name {
			flagFound = true
			res = v
			break
		}
	}
	if !flagFound {
		return res, http.StatusNotFound, errors.New(fmt.Sprintf("No record exists for name %s", name))
	}

	return res, http.StatusOK, nil
}

func UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	var e employee
	errDecode := json.NewDecoder(r.Body).Decode(&e)
	if errDecode != nil {
		// return error
		webJSONResponse(w, http.StatusBadRequest, map[string]interface{}{"error": "Falied to decode request payload"})
		return
	}
	e.Id = id // assign id value same as before as id cannot be updated by user

	var flagFound bool
	for k, v := range employeeDB {
		if v.Id == id {
			flagFound = true
			employeeDB[k] = e
			break
		}
	}
	if !flagFound {
		webJSONResponse(w, http.StatusNotFound, map[string]interface{}{"error": fmt.Sprintf("No record exists for id %d", id)})
	}

	// return
	webJSONResponse(w, http.StatusOK, map[string]interface{}{"updated": fmt.Sprintf("Employee id %d updated successfully", id)})
}

func PartialUpdateEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	var e employee
	var flagFound bool
	for k, v := range employeeDB {
		if v.Id == id {
			flagFound = true
			// fetch current values for the specific employee into e
			e = employeeDB[k]

			// update/overwrite values passed in the request payload
			errDecode := json.NewDecoder(r.Body).Decode(&e)
			if errDecode != nil {
				// return error
				webJSONResponse(w, http.StatusBadRequest, map[string]interface{}{"error": "Falied to decode request payload"})
				return
			}
			e.Id = id // assign id value same as before as id cannot be updated by user

			// replace update employee value in the employeeDB
			employeeDB[k] = e
			break
		}
	}
	if !flagFound {
		webJSONResponse(w, http.StatusNotFound, map[string]interface{}{"error": fmt.Sprintf("No record exists for id %d", id)})
	}

	// return
	webJSONResponse(w, http.StatusOK, map[string]interface{}{"updated": fmt.Sprintf("Employee id %d updated successfully", id)})
}

// DeleteEmployee to delete an employee
func DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) // *** For getting the query params
	nameORId := vars["nameORId"]
	var flagInputID bool
	id, errDecodeID := strconv.ParseInt(nameORId, 10, 64)
	if id > 0 && errDecodeID == nil {
		// input query param is a id
		flagInputID = true
		// call deleteEmployeeByID
		statusCode, err := deleteEmployeeByID(id)
		if err != nil {
			webJSONResponse(w, statusCode, map[string]interface{}{"error": err.Error()})
			return
		}
	} else {
		// input query param is a name
		// call deleteEmployeeByName
		statusCode, err := deleteEmployeeByName(nameORId)
		if err != nil {
			webJSONResponse(w, statusCode, map[string]interface{}{"error": err.Error()})
			return
		}
	}
	var msg string
	if flagInputID {
		msg = fmt.Sprintf("Employee with id %d deleted successfully", id)
	} else {
		msg = fmt.Sprintf("Employee with name %s deleted successfully", nameORId)

	}
	webJSONResponse(w, http.StatusOK, map[string]interface{}{"deleted": msg})
}

func deleteEmployeeByID(id int64) (int, error) {
	var flagFound bool
	for k, v := range employeeDB {
		if v.Id == int(id) {
			flagFound = true
			employeeDB = append(employeeDB[:k], employeeDB[k+1:]...)
			break
		}
	}
	if !flagFound {
		return http.StatusNotFound, errors.New(fmt.Sprintf("No record exists for employee id %d", id))
	}
	// return success
	return http.StatusOK, nil
}

func deleteEmployeeByName(name string) (int, error) {
	var flagFound bool
	for k, v := range employeeDB {
		if v.Name == name {
			flagFound = true
			employeeDB = append(employeeDB[:k], employeeDB[k+1:]...)
			break
		}
	}
	if !flagFound {
		return http.StatusNotFound, errors.New(fmt.Sprintf("No record exists for employee '%s'", name))
	}
	// return success
	return http.StatusOK, nil
}
