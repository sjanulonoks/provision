package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/digitalrebar/provision/backend"
)

func validateError(t *testing.T, w *httptest.ResponseRecorder, mess []string) {
	var apiErr backend.Error
	err := json.Unmarshal(w.Body.Bytes(), &apiErr)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v\n", err)
	}
	if len(apiErr.Messages) != len(mess) {
		t.Errorf("Response should be an error struct with %d message, but got: %v\n", len(mess), len(apiErr.Messages))
	} else {
		for i, m := range mess {
			if !strings.HasPrefix(apiErr.Messages[i], m) {
				t.Errorf("Response should be have start with: %s, but got: %v\n", m, apiErr.Messages[i])
			}
		}
	}
}

func validateFileInfo(t *testing.T, w *httptest.ResponseRecorder, path string, size int64) {
	var fi FileInfo
	err := json.Unmarshal(w.Body.Bytes(), &fi)
	if err != nil {
		t.Errorf("Failed to unmarshal FileInfo response: %v\n", err)
	}
	if fi.Path != path {
		t.Errorf("Path should have been %s, but was %s\n", path, fi.Path)
	}
	if fi.Size != size {
		t.Errorf("Size should have been %d, but was %d\n", size, fi.Size)
	}
}

func validateList(t *testing.T, w *httptest.ResponseRecorder, paths []string) {
	var list []string
	err := json.Unmarshal(w.Body.Bytes(), &list)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v\n", err)
	} else {
		if len(list) != len(paths) {
			t.Errorf("Response should be a list of %d, but got: %d\n", len(paths), len(list))
		} else {
			for i, p := range paths {
				if p != list[i] {
					t.Errorf("Response for list[%d] should be %s, but got: %s\n", i, p, list[i])
				}
			}
		}
	}
}

func validateNoContent(t *testing.T, w *httptest.ResponseRecorder) {
	b := w.Body.Bytes()
	if len(b) != 0 {
		t.Errorf("Message body has data! %T:%v\n", b, string(b))
	}
}

func TestFilesOps(t *testing.T) {
	localDTI := testFrontend()

	// No files present
	req, _ := http.NewRequest("GET", "/api/v3/files", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"list: error listing files: "})

	// Create a place for files.
	os.MkdirAll(tmpDir+"/files", 0755)

	req, _ = http.NewRequest("GET", "/api/v3/files", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateList(t, w, []string{})

	// Upload tests
	req, _ = http.NewRequest("POST", "/api/v3/files/jj", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusUnsupportedMediaType)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"upload: file /jj must have content-type application/octet-stream"})

	req, _ = http.NewRequest("POST", "/api/v3/files/jj", nil)
	req.Header.Set("Content-Type", "application/octet-stream")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"upload: Unable to upload /jj: missing body"})

	req, _ = http.NewRequest("POST", "/api/v3/files/jj", strings.NewReader("tempdata"))
	req.Header.Set("Content-Type", "application/octet-stream")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateFileInfo(t, w, "/jj", 8)

	req, _ = http.NewRequest("POST", "/api/v3/files/jj/jj", strings.NewReader("tempdata"))
	req.Header.Set("Content-Type", "application/octet-stream")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusConflict)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"upload: unable to create directory /jj"})

	req, _ = http.NewRequest("POST", "/api/v3/files/yy/", strings.NewReader("tempdata"))
	req.Header.Set("Content-Type", "application/octet-stream")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusForbidden)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"upload: Cannot upload a directory"})

	req, _ = http.NewRequest("POST", "/api/v3/files/kk/jj", strings.NewReader("tempdata"))
	req.Header.Set("Content-Type", "application/octet-stream")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateFileInfo(t, w, "/kk/jj", 8)

	req, _ = http.NewRequest("GET", "/api/v3/files", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateList(t, w, []string{"jj", "kk/"})

	req, _ = http.NewRequest("GET", "/api/v3/files?path=/kk", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateList(t, w, []string{"jj"})

	t.Logf("Delete testing")
	req, _ = http.NewRequest("DELETE", "/api/v3/files/", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusForbidden)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"delete: Not allowed to remove files dir"})

	req, _ = http.NewRequest("DELETE", "/api/v3/files/greg", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNotFound)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"delete: unable to delete /greg"})

	req, _ = http.NewRequest("DELETE", "/api/v3/files/kk/jj", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNoContent)
	localDTI.ValidateContentType(t, "application/json")
	validateNoContent(t, w)

	req, _ = http.NewRequest("DELETE", "/api/v3/files/jj", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNoContent)
	localDTI.ValidateContentType(t, "application/json")
	validateNoContent(t, w)

	req, _ = http.NewRequest("DELETE", "/api/v3/files/kk", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusNoContent)
	localDTI.ValidateContentType(t, "application/json")
	validateNoContent(t, w)

	req, _ = http.NewRequest("GET", "/api/v3/files", nil)
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateList(t, w, []string{})
}
