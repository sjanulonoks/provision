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

func TestBootEnvList(t *testing.T) {
	localDTI := testFrontend()

	localDTI.ListValue = nil
	validateBEList(t, localDTI, "/api/v3/bootenvs", []string{})

	localDTI.ListValue = []store.KeySaver{
		&backend.BootEnv{Name: "susan", Validation: backend.Validation{Available: true}},
		&backend.BootEnv{Name: "john", Validation: backend.Validation{Available: false}},
		&backend.BootEnv{Name: "fred", Validation: backend.Validation{Available: true}},
		&backend.BootEnv{Name: "jenny", Validation: backend.Validation{Available: false}},
		&backend.BootEnv{Name: "tess", Validation: backend.Validation{Available: true}},
	}

	// This tests the filter frontend, not the bootenvs.
	req, _ := http.NewRequest("GET", "/api/v3/bootenvs?offset=-1", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotAcceptable)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Offset cannot be negative")

	req, _ = http.NewRequest("GET", "/api/v3/bootenvs?offset=word", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotAcceptable)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Offset not valid: strconv.Atoi: parsing \"word\": invalid syntax")

	req, _ = http.NewRequest("GET", "/api/v3/bootenvs?limit=-1", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotAcceptable)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Limit cannot be negative")

	req, _ = http.NewRequest("GET", "/api/v3/bootenvs?limit=word", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotAcceptable)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Limit not valid: strconv.Atoi: parsing \"word\": invalid syntax")

	req, _ = http.NewRequest("GET", "/api/v3/bootenvs?Fred=1", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotAcceptable)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Filter not found: Fred")

	req, _ = http.NewRequest("GET", "/api/v3/bootenvs?sort=1", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotAcceptable)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Not sortable: 1")

	// Or multiple filters.
	validateBEList(t, localDTI, "/api/v3/bootenvs?offset=0&limit=1&Name=susan&Name=tess", []string{"susan"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?offset=0&limit=2&Name=susan&Name=tess", []string{"susan", "tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?offset=1&limit=2&Name=susan&Name=tess", []string{"tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs", []string{"fred", "jenny", "john", "susan", "tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?reverse=true", []string{"tess", "susan", "john", "jenny", "fred"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Between(jenny,susan)", []string{"jenny", "john", "susan"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Except(jenny,susan)", []string{"fred", "tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Between(jen,tam)", []string{"jenny", "john", "susan"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Except(jen,tam)", []string{"fred", "tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Lt(susan)", []string{"fred", "jenny", "john"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Lte(susan)", []string{"fred", "jenny", "john", "susan"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Gt(susan)", []string{"tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Gte(susan)", []string{"susan", "tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?Name=Ne(susan)", []string{"fred", "jenny", "john", "tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?sort=Available", []string{"john", "jenny", "susan", "fred", "tess"})
	validateBEList(t, localDTI, "/api/v3/bootenvs?sort=Name&sort=Available", []string{"jenny", "john", "fred", "susan", "tess"})

}

func validateBEList(t *testing.T, localDTI *LocalDTI, url string, nameOrder []string) {
	var list []backend.BootEnv
	req, _ := http.NewRequest("GET", url, nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	json.Unmarshal(w.Body.Bytes(), &list)

	if len(nameOrder) != len(list) {
		t.Errorf("%s: Response should be a list of %d, but got: %d\n", url, len(nameOrder), len(list))
		return
	}

	for i, v := range nameOrder {
		if list[i].Name != v {
			t.Errorf("%s: Response[%d] is not named %s, %v\n", url, i, v, list[i].Name)
		}
	}
}

func TestBootEnvPost(t *testing.T) {
	localDTI := testFrontend()
	t.Logf("Test invalid content type")
	localDTI.CreateValue = false
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ := http.NewRequest("POST", "/api/v3/bootenvs", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	t.Logf("Test empty body")
	localDTI.CreateValue = false
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/bootenvs", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	t.Logf("Test body not JSON")
	localDTI.CreateValue = false
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/bootenvs", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	/* GREG: handle json failure? hard to do - send a machine instead of bootenv */

	t.Logf("Test forced error")
	localDTI.CreateValue = false
	create := &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.CreateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(create)
	req, _ = http.NewRequest("POST", "/api/v3/bootenvs", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.CreateValue = true
	create = &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.CreateError = nil
	v, _ = json.Marshal(create)
	req, _ = http.NewRequest("POST", "/api/v3/bootenvs", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var be backend.BootEnv
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned BootEnv was not correct: %v %v\n", "fred", be.Name)
	}

	localDTI.CreateValue = false
	create = &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"test one"}}
	v, _ = json.Marshal(create)
	req, _ = http.NewRequest("POST", "/api/v3/bootenvs", strings.NewReader(string(v)))
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

func TestBootEnvGet(t *testing.T) {
	localDTI := testFrontend()

	localDTI.GetValue = nil
	localDTI.GetBool = false
	req, _ := http.NewRequest("GET", "/api/v3/bootenvs/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "bootenvs GET: fred: Not Found")

	localDTI.GetValue = &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.GetBool = true
	req, _ = http.NewRequest("GET", "/api/v3/bootenvs/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.BootEnv
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned BootEnv was not correct: %v %v\n", "fred", be.Name)
	}
}

/*
func TestBootEnvPatch(t *testing.T) {
	localDTI := testFrontend()

	localDTI.UpdateValue = false
	localDTI.UpdateError = nil
	req, _ := http.NewRequest("PATCH", "/api/v3/bootenvs/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotImplemented)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "bootenv patch: NOT IMPLEMENTED")
}
*/

func TestBootEnvPut(t *testing.T) {
	localDTI := testFrontend()

	localDTI.UpdateValue = false
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ := http.NewRequest("PUT", "/api/v3/bootenvs/fred", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	localDTI.UpdateValue = false
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/bootenvs/fred", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	localDTI.UpdateValue = false
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/bootenvs/fred", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	/* GREG: handle json failure? hard to do - send a machine instead of bootenv */

	localDTI.UpdateValue = false
	update := &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.UpdateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(update)
	req, _ = http.NewRequest("PUT", "/api/v3/bootenvs/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.UpdateValue = true
	update = &backend.BootEnv{Name: "kfred", Kernel: "kfred"}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(update)
	req, _ = http.NewRequest("PUT", "/api/v3/bootenvs/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "bootenvs PUT: Key change from fred to kfred not allowed")

	localDTI.UpdateValue = false
	update = &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.UpdateError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"test one"}}
	v, _ = json.Marshal(update)
	req, _ = http.NewRequest("PUT", "/api/v3/bootenvs/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "test one")

	localDTI.UpdateValue = true
	update = &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(update)
	req, _ = http.NewRequest("PUT", "/api/v3/bootenvs/fred", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.BootEnv
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Kernel != "kfred" {
		t.Errorf("Returned BootEnv was not correct: %v %v\n", "kfred", be.Kernel)
	}
}

func TestBootEnvDelete(t *testing.T) {
	localDTI := testFrontend()

	localDTI.RemoveValue = false
	localDTI.RemoveError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"should get this one"}}
	req, _ := http.NewRequest("DELETE", "/api/v3/bootenvs/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "should get this one")

	localDTI.RemoveValue = false
	localDTI.RemoveError = fmt.Errorf("this is a test: bad fred")
	req, _ = http.NewRequest("DELETE", "/api/v3/bootenvs/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.RemoveValue = true
	localDTI.RemoveError = nil
	req, _ = http.NewRequest("DELETE", "/api/v3/bootenvs/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.BootEnv
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Kernel != "kfred" {
		t.Errorf("Returned BootEnv was not correct: %v %v\n", "kfred", be.Kernel)
	}
}
