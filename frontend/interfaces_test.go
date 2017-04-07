package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/digitalrebar/provision/backend"
)

func TestInterfaceList(t *testing.T) {
	localDTI := testFrontend()

	localDTI.GIValue = nil
	localDTI.GIError = fmt.Errorf("this is a test: bad fred")
	req, _ := http.NewRequest("GET", "/api/v3/interfaces", nil)
	w := localDTI.RunTest(req)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusInternalServerError)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "interfaces list: this is a test: bad fred")

	localDTI.GIValue = nil
	localDTI.GIError = nil
	req, _ = http.NewRequest("GET", "/api/v3/interfaces", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var list []backend.Interface
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("Response should be an empty list, but got: %d\n", len(list))
	}

	localDTI.GIValue = []*backend.Interface{&backend.Interface{Name: "fred"}}
	localDTI.GIError = nil
	req, _ = http.NewRequest("GET", "/api/v3/interfaces", nil)
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

func TestInterfaceGet(t *testing.T) {
	localDTI := testFrontend()

	localDTI.GIValue = nil
	localDTI.GIError = fmt.Errorf("this is a test: bad fred")
	req, _ := http.NewRequest("GET", "/api/v3/interfaces/fred", nil)
	w := localDTI.RunTest(req)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusInternalServerError)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "interface get: this is a test: bad fred")

	localDTI.GIValue = nil
	localDTI.GIError = nil
	req, _ = http.NewRequest("GET", "/api/v3/interfaces/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "interface get: not found: fred")

	localDTI.GIValue = []*backend.Interface{&backend.Interface{Name: "ted"}}
	localDTI.GIError = nil
	req, _ = http.NewRequest("GET", "/api/v3/interfaces/fred", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "interface get: not found: fred")

	localDTI.GIValue = []*backend.Interface{&backend.Interface{Name: "fred"}}
	localDTI.GIError = nil
	req, _ = http.NewRequest("GET", "/api/v3/interfaces/fred", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var be backend.Interface
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned Interface was not correct: %v %v\n", "fred", be.Name)
	}
}
