package frontend

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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

// operation represents a valid JSON Patch operation as defined by RFC 6902
type JSONPatchOperation struct {
	// All Operations must have an Op.
	//
	// required: true
	// enum: add,remove,replace,move,copy,test
	Op string `json:"op"`

	// Path is a JSON Pointer as defined in RFC 6901
	// required: true
	Path string `json:"path"`

	// From is a JSON pointer indicating where a value should be
	// copied/moved from.  From is only used by copy and move operations.
	From string `json:"from"`

	// Value is the Value to be used for add, replace, and test operations.
	Value interface{} `json:"value"`
}

// This interface defines the pieces of the backend.DataTracker that the
// frontend needs.
type DTI interface {
	Create(store.KeySaver) (store.KeySaver, error)
	Update(store.KeySaver) (store.KeySaver, error)
	Remove(store.KeySaver) (store.KeySaver, error)
	Save(store.KeySaver) (store.KeySaver, error)
	FetchOne(store.KeySaver, string) (store.KeySaver, bool)
	FetchAll(ref store.KeySaver) []store.KeySaver

	NewBootEnv() *backend.BootEnv
	NewMachine() *backend.Machine
	NewTemplate() *backend.Template
	NewLease() *backend.Lease
	NewReservation() *backend.Reservation
	NewSubnet() *backend.Subnet
	NewUser() *backend.User

	GetInterfaces() ([]*backend.Interface, error)
}

type Frontend struct {
	FileRoot string
	MgmtApi  *gin.Engine
	ApiGroup *gin.RouterGroup
	dt       DTI
}

func NewFrontend(dt DTI, logger *log.Logger, fileRoot, devUI string) (me *Frontend) {
	mgmtApi := gin.Default()

	apiGroup := mgmtApi.Group("/api/v3")

	me = &Frontend{FileRoot: fileRoot, MgmtApi: mgmtApi, ApiGroup: apiGroup, dt: dt}

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

	// Swagger.json serve
	buf, err := embedded.Asset("assets/swagger.json")
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
		&assetfs.AssetFS{Asset: embedded.Asset, AssetDir: embedded.AssetDir, AssetInfo: embedded.AssetInfo, Prefix: "assets/swagger-ui"})

	// Server UI with flag to run from local files instead of assets
	if len(devUI) == 0 {
		mgmtApi.StaticFS("/ui",
			&assetfs.AssetFS{Asset: embedded.Asset, AssetDir: embedded.AssetDir, AssetInfo: embedded.AssetInfo, Prefix: "assets/ui"})
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
