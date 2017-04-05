package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func validateIsoInfo(t *testing.T, w *httptest.ResponseRecorder, path string, size int64) {
	var fi IsoInfo
	err := json.Unmarshal(w.Body.Bytes(), &fi)
	if err != nil {
		t.Errorf("Failed to unmarshal IsoInfo response: %v\n", err)
	}
	if fi.Path != path {
		t.Errorf("Path should have been %s, but was %s\n", path, fi.Path)
	}
	if fi.Size != size {
		t.Errorf("Size should have been %d, but was %d\n", size, fi.Size)
	}
}

func TestIsosOps(t *testing.T) {
	localDTI := testFrontend()

	// No isos present
	req, _ := http.NewRequest("GET", "/api/v3/isos", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"list: error listing isos: "})

	// Create a place for isos.
	os.MkdirAll(tmpDir+"/isos", 0755)

	req, _ = http.NewRequest("GET", "/api/v3/isos", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateList(t, w, []string{})

	// Upload tests
	req, _ = http.NewRequest("POST", "/api/v3/isos/jj", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusUnsupportedMediaType)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"upload: iso jj must have content-type application/octet-stream"})

	req, _ = http.NewRequest("POST", "/api/v3/isos/jj", nil)
	req.Header.Set("Content-Type", "application/octet-stream")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"upload: Unable to upload jj: missing body"})

	req, _ = http.NewRequest("POST", "/api/v3/isos/jj", strings.NewReader("tempdata"))
	req.Header.Set("Content-Type", "application/octet-stream")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateIsoInfo(t, w, "jj", 8)

	req, _ = http.NewRequest("GET", "/api/v3/isos", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateList(t, w, []string{"jj"})

	t.Logf("Delete testing")
	req, _ = http.NewRequest("DELETE", "/api/v3/isos/greg", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"delete: unable to delete greg"})

	req, _ = http.NewRequest("DELETE", "/api/v3/isos/jj", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNoContent)
	localDTI.ValidateContentType(t, "application/json")
	validateNoContent(t, w)

	req, _ = http.NewRequest("GET", "/api/v3/isos", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateList(t, w, []string{})
}
