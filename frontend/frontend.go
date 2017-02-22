package frontend

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
	"github.com/rackn/rocket-skates/embedded"
)

type Frontend struct {
	FileRoot    string
	MgmtApi     *gin.Engine
	ApiGroup    *gin.RouterGroup
	DataTracker *backend.DataTracker
}

func NewFrontend(dt *backend.DataTracker, logger *log.Logger, fileRoot string) (me *Frontend) {
	mgmtApi := gin.Default()

	apiGroup := mgmtApi.Group("/api/v3")

	me = &Frontend{FileRoot: fileRoot, MgmtApi: mgmtApi, ApiGroup: apiGroup, DataTracker: dt}

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
	mgmtApi.Static("/swagger-ui", "./swagger-ui/dist")

	return
}
