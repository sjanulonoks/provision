package frontend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func validateMap(t *testing.T, w *httptest.ResponseRecorder, eprefs map[string]string) {
	var prefs map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &prefs)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v\n", err)
	}
	if len(prefs) != len(eprefs) {
		t.Errorf("Response should be an error struct with %d %v prefs, but got: %d %v\n", len(eprefs), eprefs, len(prefs), prefs)
	} else {
		for k, v := range eprefs {
			v2, ok := prefs[k]
			if !ok {
				t.Errorf("prefs doesn't have expected key: %s\n", k)
			} else if v2 != v {
				t.Errorf("prefs %s doesn't match expected %s: %s\n", k, v, v2)
			}
		}
	}
}

func TestPrefsOps(t *testing.T) {
	localDTI := testFrontend()

	myPrefs := map[string]string{"fred": "joy"}
	localDTI.DefaultPrefs = myPrefs
	req, _ := http.NewRequest("GET", "/api/v3/prefs", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateMap(t, w, myPrefs)

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ := json.Marshal(map[string]string{})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"Invalid content type:"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateMap(t, w, map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{"fred": "joy"})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"Unknown Preference fred"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{"fred": "joy", "defaultBootEnv": "james"})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"Unknown Preference fred"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{"defaultBootEnv": "james"})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateMap(t, w, map[string]string{"defaultBootEnv": "james", "unknownBootEnv": "fred"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{"defaultBootEnv": "james", "unknownBootEnv": "charles"})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateMap(t, w, map[string]string{"defaultBootEnv": "james", "unknownBootEnv": "charles"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{"knownTokenTimeout": "james"})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"Preference knownTokenTimeout: strconv.Atoi: parsing \"james\": invalid syntax"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{"unknownTokenTimeout": "charles"})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusBadRequest)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateError(t, w, []string{"Preference unknownTokenTimeout: strconv.Atoi: parsing \"charles\": invalid syntax"})

	localDTI.DefaultPrefs = map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred"}
	s, _ = json.Marshal(map[string]string{"knownTokenTimeout": "50", "unknownTokenTimeout": "40"})
	req, _ = http.NewRequest("POST", "/api/v3/prefs", strings.NewReader(string(s)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusCreated)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	validateMap(t, w, map[string]string{"defaultBootEnv": "greg", "unknownBootEnv": "fred", "knownTokenTimeout": "50", "unknownTokenTimeout": "40"})
}
