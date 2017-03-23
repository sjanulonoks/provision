package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/rackn/rocket-skates/backend"
)

func TestTemplateList(t *testing.T) {
	localDTI := testFrontend()

	localDTI.ListValue = nil
	req, _ := http.NewRequest("GET", "/api/v3/templates", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var list []backend.Template
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("Response should be an empty list, but got: %d\n", len(list))
	}

	localDTI.ListValue = []store.KeySaver{&backend.Template{ID: "fred"}}
	req, _ = http.NewRequest("GET", "/api/v3/templates", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("Response should be a list of 1, but got: %d\n", len(list))
	}
	if list[0].ID != "fred" {
		t.Errorf("Response[0] is not named fred, %v\n", list[0].ID)
	}
}

func TestTemplatePost(t *testing.T) {
	localDTI := testFrontend()

	localDTI.CreateValue = nil
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ := http.NewRequest("POST", "/api/v3/templates", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	localDTI.CreateValue = nil
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/templates", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	localDTI.CreateValue = nil
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/templates", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	/* GREG: handle json failure? hard to do - send a template instead of template */

	localDTI.CreateValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.CreateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(localDTI.CreateValue)
	req, _ = http.NewRequest("POST", "/api/v3/templates", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.CreateValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.CreateError = nil
	v, _ = json.Marshal(localDTI.CreateValue)
	req, _ = http.NewRequest("POST", "/api/v3/templates", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var be backend.Template
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.ID != "fred" {
		t.Errorf("Returned Template was not correct: %v %v\n", "fred", be.ID)
	}

	localDTI.CreateValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"test one"}}
	v, _ = json.Marshal(localDTI.CreateValue)
	req, _ = http.NewRequest("POST", "/api/v3/templates", strings.NewReader(string(v)))
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

func TestTemplateGet(t *testing.T) {
	localDTI := testFrontend()

	localDTI.GetValue = nil
	localDTI.GetBool = false
	req, _ := http.NewRequest("GET", "/api/v3/templates/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "templates GET: fred: Not Found")

	localDTI.GetValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.GetBool = true
	req, _ = http.NewRequest("GET", "/api/v3/templates/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Template
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.ID != "fred" {
		t.Errorf("Returned Template was not correct: %v %v\n", "fred", be.ID)
	}
}

/*
func TestTemplatePatch(t *testing.T) {
	localDTI := testFrontend()

	localDTI.UpdateValue = nil
	localDTI.UpdateError = nil
	req, _ := http.NewRequest("PATCH", "/api/v3/templates/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotImplemented)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "template patch: NOT IMPLEMENTED")
}
*/

func TestTemplatePut(t *testing.T) {
	localDTI := testFrontend()

	localDTI.UpdateValue = nil
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ := http.NewRequest("PUT", "/api/v3/templates/fred", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	localDTI.UpdateValue = nil
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/templates/fred", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	localDTI.UpdateValue = nil
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/templates/fred", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	/* GREG: handle json failure? hard to do - send a template instead of template */

	localDTI.UpdateValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.UpdateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/templates/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.UpdateValue = &backend.Template{ID: "kfred", Contents: "kfred"}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/templates/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "templates PUT: Key change from fred to kfred not allowed")

	localDTI.UpdateValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.UpdateError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"test one"}}
	v, _ = json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/templates/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "test one")

	localDTI.UpdateValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(localDTI.UpdateValue)
	req, _ = http.NewRequest("PUT", "/api/v3/templates/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Template
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Contents != "kfred" {
		t.Errorf("Returned Template was not correct: %v %v\n", "kfred", be.Contents)
	}
}

func TestTemplateDelete(t *testing.T) {
	localDTI := testFrontend()

	localDTI.RemoveValue = nil
	localDTI.RemoveError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"should get this one"}}
	req, _ := http.NewRequest("DELETE", "/api/v3/templates/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "should get this one")

	localDTI.RemoveValue = nil
	localDTI.RemoveError = fmt.Errorf("this is a test: bad fred")
	req, _ = http.NewRequest("DELETE", "/api/v3/templates/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.RemoveValue = &backend.Template{ID: "fred", Contents: "kfred"}
	localDTI.RemoveError = nil
	req, _ = http.NewRequest("DELETE", "/api/v3/templates/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Template
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Contents != "kfred" {
		t.Errorf("Returned Template was not correct: %v %v\n", "kfred", be.Contents)
	}
}
