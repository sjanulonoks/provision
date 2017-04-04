package frontend

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/digitalrebar/go/common/store"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
	"github.com/rackn/rocket-skates/embedded"
)

// ErrorResponse is returned whenever an error occurs
// swagger:response
type ErrorResponse struct {
	//in: body
	Body backend.Error
}

// NoContentResponse is returned for deletes
// swagger:response
type NoContentResponse struct {
	//description: Nothing
}

// This interface defines the pieces of the backend.DataTracker that the
// frontend needs.
type DTI interface {
	Create(store.KeySaver) (store.KeySaver, error)
	Update(store.KeySaver) (store.KeySaver, error)
	Remove(store.KeySaver) (store.KeySaver, error)
	Save(store.KeySaver) (store.KeySaver, error)
	Patch(store.KeySaver, string, jsonpatch2.Patch) (store.KeySaver, error)
	FetchOne(store.KeySaver, string) (store.KeySaver, bool)
	FetchAll(ref store.KeySaver) []store.KeySaver

	NewBootEnv() *backend.BootEnv
	NewMachine() *backend.Machine
	NewTemplate() *backend.Template
	NewLease() *backend.Lease
	NewReservation() *backend.Reservation
	NewSubnet() *backend.Subnet
	NewUser() *backend.User
	NewParam() *backend.Param

	Pref(string) (string, error)
	Prefs() map[string]string
	SetPrefs(map[string]string) error

	GetInterfaces() ([]*backend.Interface, error)
}

type Frontend struct {
	Logger   *log.Logger
	FileRoot string
	MgmtApi  *gin.Engine
	ApiGroup *gin.RouterGroup
	dt       DTI
}

func NewFrontend(dt DTI, logger *log.Logger, fileRoot, devUI string) (me *Frontend) {
	gin.SetMode(gin.ReleaseMode)
	mgmtApi := gin.Default()

	apiGroup := mgmtApi.Group("/api/v3")

	me = &Frontend{Logger: logger, FileRoot: fileRoot, MgmtApi: mgmtApi, ApiGroup: apiGroup, dt: dt}

	me.InitBootEnvApi()
	me.InitIsoApi()
	me.InitFileApi()
	me.InitTemplateApi()
	me.InitMachineApi()
	me.InitLeaseApi()
	me.InitReservationApi()
	me.InitSubnetApi()
	me.InitUserApi()
	me.InitInterfaceApi()
	me.InitPrefApi()
	me.InitParamApi()

	// Swagger.json serve
	buf, err := embedded.Asset("swagger.json")
	if err != nil {
		logger.Fatalf("Failed to load swagger.json asset")
	}
	var f interface{}
	err = json.Unmarshal(buf, &f)
	mgmtApi.GET("/swagger.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, f)
	})

	// Server Swagger UI.
	mgmtApi.StaticFS("/swagger-ui",
		&assetfs.AssetFS{Asset: embedded.Asset, AssetDir: embedded.AssetDir, AssetInfo: embedded.AssetInfo, Prefix: "swagger-ui"})

	// Server UI with flag to run from local files instead of assets
	if len(devUI) == 0 {
		mgmtApi.StaticFS("/ui",
			&assetfs.AssetFS{Asset: embedded.Asset, AssetDir: embedded.AssetDir, AssetInfo: embedded.AssetInfo, Prefix: "ui"})
	} else {
		logger.Printf("DEV: Running UI from %s\n", devUI)
		mgmtApi.Static("/ui", devUI)
	}

	// root path, forward to UI
	mgmtApi.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/ui/")
	})

	return
}

func testContentType(c *gin.Context, ct string) bool {
	ct = strings.ToUpper(ct)
	test := strings.ToUpper(c.ContentType())

	return strings.Contains(test, ct)
}

func assureContentType(c *gin.Context, ct string) bool {
	if testContentType(c, ct) {
		return true
	}
	err := &backend.Error{Type: "API_ERROR", Code: http.StatusBadRequest}
	err.Errorf("Invalid content type: %s", c.ContentType())
	c.JSON(err.Code, err)
	return false
}

func assureDecode(c *gin.Context, val interface{}) bool {
	if !assureContentType(c, "application/json") {
		return false
	}
	err := &backend.Error{Type: "API_ERROR", Code: http.StatusBadRequest}
	marshalErr := c.Bind(&val)
	if marshalErr == nil {
		return true
	}
	err.Merge(marshalErr)
	c.JSON(err.Code, err)
	return false
}

func (f *Frontend) List(c *gin.Context, ref store.KeySaver) {
	c.JSON(http.StatusOK, f.dt.FetchAll(ref))
}

func (f *Frontend) Fetch(c *gin.Context, ref store.KeySaver, key string) {
	res, ok := f.dt.FetchOne(ref, key)
	if ok {
		c.JSON(http.StatusOK, res)
	} else {
		err := &backend.Error{
			Code:  http.StatusNotFound,
			Type:  "API_ERROR",
			Model: ref.Prefix(),
			Key:   key,
		}
		err.Errorf("%s GET: %s: Not Found", err.Model, err.Key)
		c.JSON(err.Code, err)
	}
}

func (f *Frontend) Create(c *gin.Context, val store.KeySaver) {
	if !assureDecode(c, val) {
		return
	}
	res, err := f.dt.Create(val)
	if err != nil {
		be, ok := err.(*backend.Error)
		if ok {
			c.JSON(be.Code, be)
		} else {
			c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
		}
	} else {
		c.JSON(http.StatusCreated, res)
	}
}

func (f *Frontend) Patch(c *gin.Context, ref store.KeySaver, key string) {
	patch := make(jsonpatch2.Patch, 0)
	if !assureDecode(c, &patch) {
		return
	}
	res, err := f.dt.Patch(ref, key, patch)
	if err == nil {
		c.JSON(http.StatusOK, res)
		return
	}
	ne, ok := err.(*backend.Error)
	if ok {
		c.JSON(ne.Code, ne)
	} else {
		c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
	}
}

func (f *Frontend) Update(c *gin.Context, ref store.KeySaver, key string) {
	if !assureDecode(c, ref) {
		return
	}
	if ref.Key() != key {
		err := &backend.Error{
			Code:  http.StatusBadRequest,
			Type:  "API_ERROR",
			Model: ref.Prefix(),
			Key:   key,
		}
		err.Errorf("%s PUT: Key change from %s to %s not allowed", err.Model, key, ref.Key())
		c.JSON(err.Code, err)
		return
	}
	newThing, err := f.dt.Update(ref)
	if err == nil {
		c.JSON(http.StatusOK, newThing)
		return
	}
	ne, ok := err.(*backend.Error)
	if ok {
		c.JSON(ne.Code, ne)
	} else {
		c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
	}
}

func (f *Frontend) Remove(c *gin.Context, ref store.KeySaver) {
	res, err := f.dt.Remove(ref)
	if err != nil {
		ne, ok := err.(*backend.Error)
		if ok {
			c.JSON(ne.Code, ne)
		} else {
			c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
		}
	} else {
		c.JSON(http.StatusOK, res)
	}
}
