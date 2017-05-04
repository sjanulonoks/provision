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

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/gin-gonic/gin"
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
	PatchValue   store.KeySaver
	PatchError   error
	ListValue    []store.KeySaver
	GetValue     store.KeySaver
	GetBool      bool
	GIValue      []*backend.Interface
	GIError      error
	DefaultPrefs map[string]string
	TokenValue   string
	TokenError   error
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

func (dt *LocalDTI) Patch(ref store.KeySaver, key string, patch jsonpatch2.Patch) (store.KeySaver, error) {
	return dt.PatchValue, dt.PatchError
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

func (dt *LocalDTI) Filter(ref store.KeySaver, filters ...index.Filter) ([]store.KeySaver, error) {
	idx := index.New(dt.ListValue)
	idx, err := index.All(filters...)(idx)
	return idx.Items(), err
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

func (dt *LocalDTI) GetToken(ets string) (*backend.DrpCustomClaims, error) {
	return backend.NewClaim("rocketskates", 30).Add("*", "*", "*"), nil
}
func (dt *LocalDTI) NewToken(id string, ttl int, s string, m string, a string) (string, error) {
	return dt.TokenValue, dt.TokenError
}

type TestAuthSource struct{}

var testUser *backend.User

func (tas TestAuthSource) GetUser(username string) (u *backend.User) {
	if testUser == nil {
		testUser = &backend.User{Name: username}
		testUser.PasswordHash = []byte("16384$8$1$de348bfcde8805b3b2d0435c6f4a4b96$8b649ab10cb43c7ae0717e8ccc8624aa9ebb4e76730b2e0ea4d41b91c669f234")
	}
	u = testUser
	return
}

func (dt *LocalDTI) NewBootEnv() *backend.BootEnv         { return &backend.BootEnv{} }
func (dt *LocalDTI) NewMachine() *backend.Machine         { return &backend.Machine{} }
func (dt *LocalDTI) NewProfile() *backend.Profile         { return &backend.Profile{} }
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
	localDTI.f = NewFrontend(localDTI, logger, tmpDir, devUI, &TestAuthSource{})

	return localDTI
}

func (dt *LocalDTI) RunTest(req *http.Request) *httptest.ResponseRecorder {
	// BASIC AUTH TESTING: req.SetBasicAuth("rocketskates", "r0cketsk8ts")
	// BEARER AUTH TESTING:
	req.Header.Add("Authorization", "Bearer MyFakeToken")
	dt.w = httptest.NewRecorder()
	dt.f.MgmtApi.ServeHTTP(dt.w, req)
	return dt.w
}

func (dt *LocalDTI) ValidateCode(t *testing.T, c int) {
	if dt.w.Code != c {
		t.Errorf("Response should be %v, was: %v", c, dt.w.Code)
	} else {
		t.Logf("Got expected code %d", c)
	}
}

func (dt *LocalDTI) ValidateContentType(t *testing.T, ct string) {
	if dt.w.HeaderMap.Get("Content-Type") != ct {
		t.Errorf("Content-Type should be %v, was %v", ct, dt.w.HeaderMap.Get("Content-Type"))
	} else {
		t.Logf("Got expected content-type: %s", ct)
	}
}

func (dt *LocalDTI) ValidateError(t *testing.T, ap string, mess string) {
	var err backend.Error
	buf := dt.w.Body.Bytes()
	lerr := json.Unmarshal(buf, &err)
	if lerr != nil {
		t.Errorf("Response should be valid error struct: %v: %v\n", lerr, err)
	} else {
		t.Logf("For response body: %s\n", string(buf))
		t.Logf("Got error log: %#v", err)
	}
	if err.Type != ap {
		t.Errorf("Error type should be: %v, but is %v\n", ap, err.Type)
	}
	if len(err.Messages) != 1 {
		t.Errorf("Error messages should be length one, but is: %v\n", len(err.Messages))
	} else {
		if err.Messages[0] != mess {
			t.Errorf("Error mess should be: %v, but is %v\n", mess, err.Messages[0])
		}
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
	if !bytes.Contains(uibody, []byte("<title>Digital Rebar: Provision</title>")) {
		t.Errorf("Digital Rebar: Provision Title Missing %v\n", uibody)
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
	if !bytes.Contains(uibody, []byte("<title>Test DRP</title>")) {
		t.Errorf("Digital Rebar UI Dev Mode Not Working! %v\n", uibody)
	}
}

// GREG: Test DefaultAuthSource

var tmpDir string

func TestMain(m *testing.M) {
	var err error
	tmpDir, err = ioutil.TempDir("", "frontend-")
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	ret := m.Run()
	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	os.Exit(ret)
}
