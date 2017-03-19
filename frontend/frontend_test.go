package frontend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

type LocalDTI struct {
	CreateValue  store.KeySaver
	CreateError  error
	UpdateValue  store.KeySaver
	UpdateError  error
	SaveValue    store.KeySaver
	SaveError    error
	RemoveValue  store.KeySaver
	RemoveError  error
	ListValue    []store.KeySaver
	GetValue     store.KeySaver
	GetBool      bool
	GIValue      []*backend.Interface
	GIError      error
	DefaultPrefs map[string]string
	w            *httptest.ResponseRecorder
	f            *Frontend
}

func (dt *LocalDTI) Create(store.KeySaver) (store.KeySaver, error) {
	return dt.CreateValue, dt.CreateError
}
func (dt *LocalDTI) Update(store.KeySaver) (store.KeySaver, error) {
	return dt.UpdateValue, dt.UpdateError
}
func (dt *LocalDTI) Remove(store.KeySaver) (store.KeySaver, error) {
	return dt.RemoveValue, dt.RemoveError
}
func (dt *LocalDTI) Save(store.KeySaver) (store.KeySaver, error) {
	return dt.SaveValue, dt.SaveError
}
func (dt *LocalDTI) FetchOne(store.KeySaver, string) (store.KeySaver, bool) {
	return dt.GetValue, dt.GetBool
}
func (dt *LocalDTI) FetchAll(ref store.KeySaver) []store.KeySaver {
	return dt.ListValue
}
func (dt *LocalDTI) GetInterfaces() ([]*backend.Interface, error) {
	return dt.GIValue, dt.GIError
}

func (dt *LocalDTI) Pref(name string) (string, error) {
	res, ok := dt.DefaultPrefs[name]
	if ok {
		return res, nil
	}
	return "", fmt.Errorf("Missing pref %s", name)
}

func (dt *LocalDTI) Prefs() map[string]string {
	return dt.DefaultPrefs
}

func (dt *LocalDTI) SetPrefs(prefs map[string]string) error {
	for name, val := range prefs {
		dt.DefaultPrefs[name] = val
	}
	return nil
}

func (dt *LocalDTI) NewBootEnv() *backend.BootEnv         { return &backend.BootEnv{} }
func (dt *LocalDTI) NewMachine() *backend.Machine         { return &backend.Machine{} }
func (dt *LocalDTI) NewTemplate() *backend.Template       { return &backend.Template{} }
func (dt *LocalDTI) NewLease() *backend.Lease             { return &backend.Lease{} }
func (dt *LocalDTI) NewReservation() *backend.Reservation { return &backend.Reservation{} }
func (dt *LocalDTI) NewSubnet() *backend.Subnet           { return &backend.Subnet{} }
func (dt *LocalDTI) NewUser() *backend.User               { return &backend.User{} }

func testFrontend() *LocalDTI {
	return testFrontendDev("")
}

func testFrontendDev(devUI string) *LocalDTI {
	gin.SetMode(gin.ReleaseMode)

	localDTI := &LocalDTI{}
	logger := log.New(os.Stderr, "bootenv-test", log.LstdFlags|log.Lmicroseconds|log.LUTC)
	localDTI.f = NewFrontend(localDTI, logger, ".", devUI)

	return localDTI
}

func (dt *LocalDTI) RunTest(req *http.Request) *httptest.ResponseRecorder {
	dt.w = httptest.NewRecorder()
	dt.f.MgmtApi.ServeHTTP(dt.w, req)
	return dt.w
}

func (dt *LocalDTI) ValidateCode(t *testing.T, c int) {
	if dt.w.Code != c {
		t.Errorf("Response should be %v, was: %v", c, dt.w.Code)
	}
}

func (dt *LocalDTI) ValidateContentType(t *testing.T, ct string) {
	if dt.w.HeaderMap.Get("Content-Type") != ct {
		t.Errorf("Content-Type should be %v, was %v", ct, dt.w.HeaderMap.Get("Content-Type"))
	}
}

func (dt *LocalDTI) ValidateError(t *testing.T, ap string, mess string) {
	var err backend.Error
	lerr := json.Unmarshal(dt.w.Body.Bytes(), &err)
	if lerr != nil {
		t.Errorf("Response should be valid error struct: %v: %v\n", lerr, err)
	}
	if err.Type != ap {
		t.Errorf("Error type should be: %v, but is %v\n", ap, err.Type)
	}
	if len(err.Messages) != 1 {
		t.Errorf("Error messages should be length one, but is: %v\n", len(err.Messages))
	}
	if err.Messages[0] != mess {
		t.Errorf("Error mess should be: %v, but is %v\n", mess, err.Messages[0])
	}
}

func TestSwaggerPieces(t *testing.T) {
	localDTI := testFrontend()

	req, _ := http.NewRequest("GET", "/swagger.json", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "application/json; charset=utf-8")
	var swagger map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &swagger)
	if len(swagger) == 0 {
		t.Errorf("Response should not be an empty set, but got: %d\n", len(swagger))
	}
	s := swagger["swagger"].(string)
	if s != "2.0" {
		t.Errorf("Swagger version should be 2.0: %v\n", s)
	}
}

func TestRoot(t *testing.T) {
	localDTI := testFrontend()

	req, _ := http.NewRequest("GET", "/", nil)
	localDTI.RunTest(req)
	localDTI.ValidateCode(t, 302)
}

func TestUIBase(t *testing.T) {
	localDTI := testFrontend()

	req, _ := http.NewRequest("GET", "/ui/", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "text/html; charset=utf-8")
	uibody, _ := ioutil.ReadAll(w.Body)
	if len(uibody) == 0 {
		t.Errorf("Response should not be an empty set, but got: %d\n", len(uibody))
	}
	if !bytes.Contains(uibody, []byte("<title>Rocket Skates</title>")) {
		t.Errorf("Rocket Skates Title Missing %v\n", uibody)
	}
}

func TestUIDev(t *testing.T) {
	localDTI := testFrontendDev("../test-data/ui")
	req, _ := http.NewRequest("GET", "/ui/", nil)
	w := localDTI.RunTest(req)
	localDTI.ValidateCode(t, http.StatusOK)
	localDTI.ValidateContentType(t, "text/html; charset=utf-8")
	uibody, _ := ioutil.ReadAll(w.Body)
	if len(uibody) == 0 {
		t.Errorf("Response should not be an empty set, but got: %d\n", len(uibody))
	}
	if !bytes.Contains(uibody, []byte("<title>Test Skates</title>")) {
		t.Errorf("Rocket Skates Title Missing %v\n", uibody)
	}
}
