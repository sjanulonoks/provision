package frontend

import (
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

type Frontend struct {
	BasePath    string
	FileRoot    string
	MgmtApi     *gin.Engine
	DataTracker *backend.DataTracker
}

func NewFrontend(dt *backend.DataTracker, basePath, fileRoot string) (me *Frontend, err error) {
	mgmtApi := gin.Default()

	err = nil
	me = &Frontend{BasePath: basePath, FileRoot: fileRoot, MgmtApi: mgmtApi, DataTracker: dt}

	me.InitBootEnvApi()
	me.InitIsoApi()
	me.InitFileApi()
	me.InitTemplateApi()
	me.InitMachineApi()

	return
}
