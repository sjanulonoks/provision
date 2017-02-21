package frontend

import (
	"net/http"

	"github.com/rackn/rocket-skates/backend"
)

func initMachineRoutes(dt *backend.DataTracker) {
	mgmtApi.GET(basePath+"/machines",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, dt.AsMachines(dt.FetchAll(dt.NewMachine())))
		})
	mgmtApi.POST(basePath+"/machines",
		func(c *gin.Context) {
			b := dt.NewMachine()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, NewError(err.Error()))
			}
			ok, err := dt.Create(b)
			if err != nil {
				c.JSON(http.StatusBadRequest, NewError(err.Error()))
			} else {
				c.JSON(http.StatusCreated, b)
			}
		})
	mgmtApi.GET(basePath+"/machines/:name",
		func(c *gin.Context) {
			res, ok := dt.FetchOne(dt.NewMachine(), c.Param(`name`))
			if ok {
				c.JSON(http.StatusOK, dt.AsMachine(res))
			} else {
				c.JSON(http.StatusNotFound, err)
			}
		})
	mgmtApi.PATCH(basePath+"/machines/:name",
		func(c *gin.Context) {
			//			updateThing(c, &Machine{Name: c.Param(`name`)}, &Machine{})
		})
	mgmtApi.PUT(basePath+"/machines/:name",
		func(c *gin.Context) {
			b := dt.NewMachine()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, NewError(err.Error()))
			}
			if b.Name != c.Param(`name`) {
				c.JSON(http.StatusBadRequest, NewError(err.Error()))
			}
			ok, err := dt.Update(b)
			if !ok && err != nil {
				c.JSON(http.StatusNotFound, err)
			} else if !ok {
				c.JSON(http.StatusNotFound, err)
			} else {
				c.JSON(http.StatusOK, b)
			}
		})
	mgmtApi.DELETE(basePath+"/machines/:name",
		func(c *gin.Context) {
			b := dt.NewMachine()
			b.Name = c.Param(`name`)
			ok, err := dt.Remove(b)
			if !ok && err != nil {
				c.JSON(http.StatusNotFound, err)
			} else if !ok {
				c.JSON(http.StatusNotFound, err)
			} else {
				c.JSON(http.StatusNoContent)
			}
		})
}

/*
func BootenvPatch(params machines.PatchBootenvParams, p *models.Principal) middleware.Responder {
	newThing := NewBootenv(params.Name)
	patch, _ := json.Marshal(params.Body)
	item, err := patchThing(newThing, patch)
	if err != nil {
		if err.Code == http.StatusNotFound {
			return machines.NewPatchBootenvNotFound().WithPayload(err)
		}
		if err.Code == http.StatusConflict {
			return machines.NewPatchBootenvConflict().WithPayload(err)
		}
		return machines.NewPatchBootenvExpectationFailed().WithPayload(err)
	}
	original, ok := item.(models.BootenvOutput)
	if !ok {
		e := NewError(http.StatusInternalServerError, "Could not marshal bootenv")
		return machines.NewPatchBootenvInternalServerError().WithPayload(e)
	}
	return machines.NewPatchBootenvOK().WithPayload(&original)
}
*/
