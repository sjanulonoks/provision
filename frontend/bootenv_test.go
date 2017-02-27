package frontend

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/rackn/rocket-skates/backend"
)

func TestBootEnvList(t *testing.T) {
	localDTI := testFrontend()

	localDTI.ListValue = nil
	req, _ := http.NewRequest("GET", "/api/v3/bootenvs", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var list []backend.BootEnv
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Errorf("Response should be an empty list, but got: %d\n", len(list))
	}

	localDTI.ListValue = []store.KeySaver{&backend.BootEnv{Name: "fred"}}
	req, _ = http.NewRequest("GET", "/api/v3/bootenvs", nil)
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

func TestBootEnvPost(t *testing.T) {
	localDTI := testFrontend()

	localDTI.ListValue = nil
	req, _ := http.NewRequest("POST", "/api/v3/bootenvs", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	localDTI.ValidateError(t, "API_ERROR", "missing form body")

	localDTI.CreateValue = &backend.BootEnv{Name: "fred", Kernel: "kfred"}
	localDTI.CreateError = nil
	v, _ := json.Marshal(localDTI.CreateValue)
	req, _ = http.NewRequest("POST", "/api/v3/bootenvs", strings.NewReader(string(v)))
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")

	var be backend.Machine
	json.Unmarshal(w.Body.Bytes(), &be)
	if be.Name != "fred" {
		t.Errorf("Returned BootEnv was not correct: %v %v\n", "fred", be.Name)
	}
}
