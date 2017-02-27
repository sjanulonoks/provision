package frontend

import (
	"encoding/json"
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
	CreateValue store.KeySaver
	CreateError error
	UpdateValue bool
	UpdateError error
	SaveValue   store.KeySaver
	SaveError   error
	RemoveValue store.KeySaver
	RemoveError error
	ListValue   []store.KeySaver
	GetValue    store.KeySaver
	GetBool     bool
	w           *httptest.ResponseRecorder
	f           *Frontend
}

func (dt *LocalDTI) Create(store.KeySaver) (store.KeySaver, error) {
	return dt.CreateValue, dt.CreateError
}
func (dt *LocalDTI) Update(store.KeySaver) (bool, error) {
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

func (dt *LocalDTI) NewBootEnv() *backend.BootEnv   { return &backend.BootEnv{} }
func (dt *LocalDTI) NewMachine() *backend.Machine   { return &backend.Machine{} }
func (dt *LocalDTI) NewTemplate() *backend.Template { return &backend.Template{} }

func testFrontend() *LocalDTI {
	gin.SetMode(gin.ReleaseMode)

	localDTI := &LocalDTI{}
	logger := log.New(os.Stderr, "bootenv-test", log.LstdFlags|log.Lmicroseconds|log.LUTC)
	localDTI.f = NewFrontend(localDTI, logger, ".")

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
