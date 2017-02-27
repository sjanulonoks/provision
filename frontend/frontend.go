package frontend

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
	"github.com/rackn/rocket-skates/embedded"
)

// This interface defines the pieces of the backend.DataTracker that the
// frontend needs.
type DTI interface {
	Create(store.KeySaver) (store.KeySaver, error)
	Update(store.KeySaver) (bool, error)
	Remove(store.KeySaver) (store.KeySaver, error)
	Save(store.KeySaver) (store.KeySaver, error)
	FetchOne(store.KeySaver, string) (store.KeySaver, bool)
	FetchAll(ref store.KeySaver) []store.KeySaver

	NewBootEnv() *backend.BootEnv
	NewMachine() *backend.Machine
	NewTemplate() *backend.Template
}

type Frontend struct {
	FileRoot string
	MgmtApi  *gin.Engine
	ApiGroup *gin.RouterGroup
	dt       DTI
}

func NewFrontend(dt DTI, logger *log.Logger, fileRoot string) (me *Frontend) {
	mgmtApi := gin.Default()

	apiGroup := mgmtApi.Group("/api/v3")

	me = &Frontend{FileRoot: fileRoot, MgmtApi: mgmtApi, ApiGroup: apiGroup, dt: dt}

	me.InitBootEnvApi()
	me.InitIsoApi()
	me.InitFileApi()
	me.InitTemplateApi()
	me.InitMachineApi()

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

	return
}
