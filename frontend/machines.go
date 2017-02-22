package frontend

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

func (f *Frontend) InitMachineApi() {
	f.ApiGroup.GET("/machines",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, backend.AsMachines(f.DataTracker.FetchAll(f.DataTracker.NewMachine())))
		})
	f.ApiGroup.POST("/machines",
		func(c *gin.Context) {
			b := f.DataTracker.NewMachine()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
			}
			nb, err := f.DataTracker.Create(b)
			if err != nil {
				c.JSON(http.StatusBadRequest, err)
			} else {
				c.JSON(http.StatusCreated, nb)
			}
		})
	f.ApiGroup.GET("/machines/:name",
		func(c *gin.Context) {
			res, ok := f.DataTracker.FetchOne(f.DataTracker.NewMachine(), c.Param(`name`))
			if ok {
				c.JSON(http.StatusOK, backend.AsMachine(res))
			} else {
				c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusNotFound,
					fmt.Sprintf("machines: Not Found: %v", c.Param(`name`))))
			}
		})
	f.ApiGroup.PATCH("/machines/:name",
		func(c *gin.Context) {
			//			updateThing(c, &Machine{Name: c.Param(`name`)}, &Machine{})
		})
	f.ApiGroup.PUT("/machines/:name",
		func(c *gin.Context) {
			b := f.DataTracker.NewMachine()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
			}
			if b.Name != c.Param(`name`) {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest,
					fmt.Sprintf("machines: Can not change name: %v -> %v", c.Param(`name`), b.Name)))
			}
			nb, err := f.DataTracker.Update(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err) // GREG: COde
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})
	f.ApiGroup.DELETE("/machines/:name",
		func(c *gin.Context) {
			b := f.DataTracker.NewMachine()
			b.Name = c.Param(`name`)
			_, err := f.DataTracker.Remove(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err) // GREG: Code
			} else {
				c.JSON(http.StatusNoContent, nil)
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
