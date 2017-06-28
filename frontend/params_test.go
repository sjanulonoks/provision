package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend"
)

func TestParamList(t *testing.T) {
	localDTI := testFrontend()

	localDTI.ListValue = nil
	req, _ := http.NewRequest("GET", "/api/v3/params", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var list []backend.Param
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("Response should be an empty list, but got: %d\n", len(list))
	}

	localDTI.ListValue = []store.KeySaver{&backend.Param{Name: "fred"}}
	req, _ = http.NewRequest("GET", "/api/v3/params", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("Response should be a list of 1, but got: %d\n", len(list))
	}
	if list[0].Name != "fred" {
		t.Errorf("Response[0] is not named fred, %v\n", list[0].Name)
	}
}

func TestParamPost(t *testing.T) {
	localDTI := testFrontend()

	localDTI.CreateValue = nil
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ := http.NewRequest("POST", "/api/v3/params", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	localDTI.CreateValue = nil
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/params", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	localDTI.CreateValue = nil
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/params", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	/* GREG: handle json failure? bad IP */

	localDTI.CreateValue = &backend.Param{Name: "fred"}
	localDTI.CreateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(localDTI.CreateValue)
	req, _ = http.NewRequest("POST", "/api/v3/params", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.CreateValue = &backend.Param{Name: "fred"}
	localDTI.CreateError = nil
	v, _ = json.Marshal(localDTI.CreateValue)
	req, _ = http.NewRequest("POST", "/api/v3/params", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var be backend.Param
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned Param was not correct: %v %v\n", "fred", be.Name)
	}

	localDTI.CreateValue = &backend.Param{Name: "fred"}
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"test one"}}
	v, _ = json.Marshal(localDTI.CreateValue)
	req, _ = http.NewRequest("POST", "/api/v3/params", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var berr backend.Error
	json.Unmarshal(w.Body.Bytes(), &berr)
	if berr.Messages[0] != "test one" {
		t.Errorf("Returned Error was not correct: %v %v\n", "test one", berr.Messages[0])
	}
}

func TestParamGet(t *testing.T) {
	localDTI := testFrontend()

	localDTI.GetValue = nil
	localDTI.GetBool = false
	req, _ := http.NewRequest("GET", "/api/v3/params/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "params GET: fred: Not Found")

	localDTI.GetValue = &backend.Param{Name: "fred"}
	localDTI.GetBool = true
	req, _ = http.NewRequest("GET", "/api/v3/params/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Param
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned Param was not correct: %v %v\n", "fred", be.Name)
	}
}

/*
func TestParamPatch(t *testing.T) {
	localDTI := testFrontend()

	localDTI.UpdateValue = nil
	localDTI.UpdateError = nil
	req, _ := http.NewRequest("PATCH", "/api/v3/params/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotImplemented)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "param patch: NOT IMPLEMENTED")
}
*/

func TestParamPut(t *testing.T) {
	localDTI := testFrontend()

	localDTI.UpdateValue = nil
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ := http.NewRequest("PUT", "/api/v3/params/fred", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	localDTI.UpdateValue = nil
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/params/fred", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	localDTI.UpdateValue = nil
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/params/fred", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	localDTI.UpdateValue = nil
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/params/george", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	/* GREG: handle json failure? hard to do - send a machine instead of param */

	localDTI.UpdateValue = &backend.Param{Name: "fred"}
	localDTI.UpdateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/params/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.UpdateValue = &backend.Param{Name: "george"}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/params/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "params PUT: Key change from fred to george not allowed")

	localDTI.UpdateValue = &backend.Param{Name: "fred"}
	localDTI.UpdateError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"test one"}}
	v, _ = json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/params/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "test one")

	localDTI.UpdateValue = &backend.Param{Name: "fred"}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/params/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Param
	json.Unmarshal(w.Body.Bytes(), &be)
}

func TestParamDelete(t *testing.T) {
	localDTI := testFrontend()

	localDTI.RemoveValue = nil
	localDTI.RemoveError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"should get this one"}}
	req, _ := http.NewRequest("DELETE", "/api/v3/params/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "should get this one")

	localDTI.RemoveValue = nil
	localDTI.RemoveError = fmt.Errorf("this is a test: bad fred")
	req, _ = http.NewRequest("DELETE", "/api/v3/params/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.RemoveValue = &backend.Param{Name: "fred"}
	localDTI.RemoveError = nil
	req, _ = http.NewRequest("DELETE", "/api/v3/params/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Param
	json.Unmarshal(w.Body.Bytes(), &be)
}
