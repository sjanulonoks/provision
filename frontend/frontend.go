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
	me.InitIsoAPI()
	me.InitFileAPI()

	// machine methods
	mgmtApi.GET("/machines",
		func(c *gin.Context) {
			listThings(c, &Machine{})
		})
	mgmtApi.POST("/machines",
		func(c *gin.Context) {
			createThing(c, &Machine{})
		})
	mgmtApi.GET("/machines/:name", func(c *gin.Context) {
		getThing(c, popMachine(c.Param(`name`)))
	})
	mgmtApi.PATCH("/machines/:name",
		func(c *gin.Context) {
			updateThing(c, popMachine(c.Param(`name`)), &Machine{})
		})
	mgmtApi.DELETE("/machines/:name",
		func(c *gin.Context) {
			deleteThing(c, popMachine(c.Param(`name`)))
		})

	// template methods
	mgmtApi.GET("/templates",
		func(c *gin.Context) {
			listThings(c, &Template{})
		})
	mgmtApi.POST("/templates",
		func(c *gin.Context) {
			createThing(c, &Template{})
		})
	mgmtApi.POST("/templates/:uuid", createTemplate)
	mgmtApi.GET("/templates/:uuid",
		func(c *gin.Context) {
			getThing(c, &Template{UUID: c.Param(`uuid`)})
		})
	mgmtApi.PATCH("/templates/:uuid",
		func(c *gin.Context) {
			updateThing(c, &Template{UUID: c.Param(`uuid`)}, &Template{})
		})
	mgmtApi.DELETE("/templates/:uuid",
		func(c *gin.Context) {
			deleteThing(c, &Template{UUID: c.Param(`uuid`)})
		})

	return
}
