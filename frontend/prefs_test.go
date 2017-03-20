package frontend

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPrefsList(t *testing.T) {
	dti := testFrontend()
	req, _ := http.NewRequest("GET", "/api/v3/prefs", nil)
	resp := dti.RunTest(req)
	dti.ValidateCode(t, http.StatusOK)
	dti.ValidateContentType(t, "application/json; charset=utf-8")
	var prefs map[string]string
	json.Unmarshal(resp.Body.Bytes(), &prefs)
	if len(prefs) != 0 {
		t.Errorf("Should have gotten no preferences")
	}
}
