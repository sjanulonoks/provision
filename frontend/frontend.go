package frontend

import (
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

type Frontend struct {
	FileRoot    string
	MgmtApi     *gin.Engine
	ApiGroup    *gin.RouterGroup
	DataTracker *backend.DataTracker
}

func NewFrontend(dt *backend.DataTracker, fileRoot string) (me *Frontend) {
	mgmtApi := gin.Default()

	apiGroup := mgmtApi.Group("/api/v3")

	me = &Frontend{FileRoot: fileRoot, MgmtApi: mgmtApi, ApiGroup: apiGroup, DataTracker: dt}

	me.InitBootEnvApi()
	me.InitIsoApi()
	me.InitFileApi()
	me.InitTemplateApi()
	me.InitMachineApi()

	return
}
