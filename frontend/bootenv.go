package frontend

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

func (f *Frontend) InitBootEnvApi() {
	f.MgmtApi.GET(f.BasePath+"/bootenvs",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, backend.AsBootEnvs(f.DataTracker.FetchAll(f.DataTracker.NewBootEnv())))
		})
	f.MgmtApi.POST(f.BasePath+"/bootenvs",
		func(c *gin.Context) {
			b := f.DataTracker.NewBootEnv()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, err)
			}
			nb, err := f.DataTracker.Create(b)
			if err != nil {
				c.JSON(http.StatusBadRequest, err)
			} else {
				c.JSON(http.StatusCreated, nb)
			}
		})
	f.MgmtApi.GET(f.BasePath+"/bootenvs/:name",
		func(c *gin.Context) {
			res, ok := f.DataTracker.FetchOne(f.DataTracker.NewBootEnv(), c.Param(`name`))
			if ok {
				c.JSON(http.StatusOK, backend.AsBootEnv(res))
			} else {
				c.JSON(http.StatusNotFound, nil) // GREG: Fix
			}
		})
	f.MgmtApi.PATCH(f.BasePath+"/bootenvs/:name",
		func(c *gin.Context) {
			//			updateThing(c, &BootEnv{Name: c.Param(`name`)}, &BootEnv{})
		})
	f.MgmtApi.PUT(f.BasePath+"/bootenvs/:name",
		func(c *gin.Context) {
			b := f.DataTracker.NewBootEnv()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, err)
			}
			if b.Name != c.Param(`name`) {
				c.JSON(http.StatusBadRequest, nil) // GREG: Fix
			}
			nb, err := f.DataTracker.Update(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err)
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})
	f.MgmtApi.DELETE(f.BasePath+"/bootenvs/:name",
		func(c *gin.Context) {
			b := f.DataTracker.NewBootEnv()
			b.Name = c.Param(`name`)
			_, err := f.DataTracker.Remove(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err)
			} else {
				c.JSON(http.StatusNoContent, nil)
			}
		})
}

/*
func BootenvPatch(params bootenvs.PatchBootenvParams, p *models.Principal) middleware.Responder {
	newThing := NewBootenv(params.Name)
	patch, _ := json.Marshal(params.Body)
	item, err := patchThing(newThing, patch)
	if err != nil {
		if err.Code == http.StatusNotFound {
			return bootenvs.NewPatchBootenvNotFound().WithPayload(err)
		}
		if err.Code == http.StatusConflict {
			return bootenvs.NewPatchBootenvConflict().WithPayload(err)
		}
		return bootenvs.NewPatchBootenvExpectationFailed().WithPayload(err)
	}
	original, ok := item.(models.BootenvOutput)
	if !ok {
		e := NewError(http.StatusInternalServerError, "Could not marshal bootenv")
		return bootenvs.NewPatchBootenvInternalServerError().WithPayload(e)
	}
	return bootenvs.NewPatchBootenvOK().WithPayload(&original)
}
*/
