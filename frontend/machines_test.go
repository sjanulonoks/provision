package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend"
	"github.com/pborman/uuid"
)

func TestMachineList(t *testing.T) {
	localDTI := testFrontend()

	localDTI.ListValue = nil
	req, _ := http.NewRequest("GET", "/api/v3/machines", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var list []backend.Machine
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("Response should be an empty list, but got: %d\n", len(list))
	}

	localDTI.ListValue = []store.KeySaver{&backend.Machine{Name: "fred"}}
	req, _ = http.NewRequest("GET", "/api/v3/machines", nil)
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

func TestMachinePost(t *testing.T) {
	localDTI := testFrontend()

	localDTI.CreateValue = false
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ := http.NewRequest("POST", "/api/v3/machines", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	localDTI.CreateValue = false
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/machines", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	localDTI.CreateValue = false
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"should not get this"}}
	req, _ = http.NewRequest("POST", "/api/v3/machines", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	/* GREG: handle json failure? hard to do - send a machine instead of machine */
	localDTI.CreateValue = false
	create := &backend.Machine{Name: "fred", BootEnv: "kfred"}
	localDTI.CreateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(create)
	req, _ = http.NewRequest("POST", "/api/v3/machines", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.CreateValue = true
	create = &backend.Machine{Name: "fred", BootEnv: "kfred"}
	localDTI.CreateError = nil
	v, _ = json.Marshal(create)
	req, _ = http.NewRequest("POST", "/api/v3/machines", strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var be backend.Machine
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned Machine was not correct: %v %v\n", "fred", be.Name)
	}

	localDTI.CreateValue = false
	create = &backend.Machine{Name: "fred", BootEnv: "kfred"}
	localDTI.CreateError = &backend.Error{Code: 23, Messages: []string{"test one"}}
	v, _ = json.Marshal(create)
	req, _ = http.NewRequest("POST", "/api/v3/machines", strings.NewReader(string(v)))
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

// TODO: TEST MACHINE NAME Duplication error

func TestMachineGet(t *testing.T) {
	localDTI := testFrontend()

	localDTI.GetValue = nil
	localDTI.GetBool = false
	req, _ := http.NewRequest("GET", "/api/v3/machines/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "machines GET: fred: Not Found")

	localDTI.GetValue = &backend.Machine{Name: "fred", BootEnv: "kfred"}
	localDTI.GetBool = true
	req, _ = http.NewRequest("GET", "/api/v3/machines/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Machine
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned Machine was not correct: %v %v\n", "fred", be.Name)
	}
}

/*
func TestMachinePatch(t *testing.T) {
	localDTI := testFrontend()

	localDTI.UpdateValue = true
	localDTI.UpdateError = nil
	req, _ := http.NewRequest("PATCH", "/api/v3/machines/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotImplemented)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "machine patch: NOT IMPLEMENTED")
}
*/

func TestMachinePut(t *testing.T) {
	localDTI := testFrontend()
	goodUUID := uuid.NewRandom()
	badUUID := uuid.NewRandom()
	goodPath := fmt.Sprintf("/api/v3/machines/%s", goodUUID)
	kcm := fmt.Sprintf("machines PUT: Key change from %s to %s not allowed", goodUUID, badUUID)

	localDTI.UpdateValue = false
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ := http.NewRequest("PUT", "/api/v3/machines/fred", nil)
	req.Header.Set("Content-Type", "text/html")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "Invalid content type: text/html")

	localDTI.UpdateValue = false
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/machines/fred", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "EOF")

	localDTI.UpdateValue = false
	localDTI.UpdateError = &backend.Error{Code: 23, Messages: []string{"should not get this one"}}
	req, _ = http.NewRequest("PUT", "/api/v3/machines/fred", strings.NewReader("asgasgd"))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "invalid character 'a' looking for beginning of value")

	/* GREG: handle json failure? hard to do - send a machine instead of machine */

	localDTI.UpdateValue = false
	update := &backend.Machine{Name: "fred", BootEnv: "kfred", Uuid: goodUUID}
	localDTI.UpdateError = fmt.Errorf("this is a test: bad fred")
	v, _ := json.Marshal(update)
	req, _ = http.NewRequest("PUT", goodPath, strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.UpdateValue = true
	update = &backend.Machine{Name: "fred", BootEnv: "kfred", Uuid: badUUID}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(update)
	req, _ = http.NewRequest("PUT", goodPath, strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", kcm)

	localDTI.UpdateValue = false
	update = &backend.Machine{Name: "fred", BootEnv: "kfred", Uuid: goodUUID}
	localDTI.UpdateError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"test one"}}
	v, _ = json.Marshal(update)
	req, _ = http.NewRequest("PUT", goodPath, strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "test one")

	localDTI.UpdateValue = true
	update = &backend.Machine{Name: "fred", BootEnv: "kfred", Uuid: goodUUID}
	localDTI.UpdateError = nil
	v, _ = json.Marshal(update)
	req, _ = http.NewRequest("PUT", goodPath, strings.NewReader(string(v)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Machine
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.BootEnv != "kfred" {
		t.Errorf("Returned Machine was not correct: %v %v\n", "kfred", be.BootEnv)
	}
}

func TestMachineDelete(t *testing.T) {
	localDTI := testFrontend()

	localDTI.RemoveValue = false
	localDTI.RemoveError = &backend.Error{Code: 23, Type: "API_ERROR", Messages: []string{"should get this one"}}
	req, _ := http.NewRequest("DELETE", "/api/v3/machines/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 23)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "should get this one")

	localDTI.RemoveValue = false
	localDTI.RemoveError = fmt.Errorf("this is a test: bad fred")
	req, _ = http.NewRequest("DELETE", "/api/v3/machines/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "this is a test: bad fred")

	localDTI.RemoveValue = true
	localDTI.RemoveError = nil
	req, _ = http.NewRequest("DELETE", "/api/v3/machines/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Machine
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.BootEnv != "kfred" {
		t.Errorf("Returned Machine was not correct: %v %v\n", "kfred", be.BootEnv)
	}
}
