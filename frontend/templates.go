package frontend

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

func (f *Frontend) InitTemplateApi() {
	f.ApiGroup.GET("/templates",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, backend.AsTemplates(f.DataTracker.FetchAll(f.DataTracker.NewTemplate())))
		})
	f.ApiGroup.POST("/templates",
		func(c *gin.Context) {
			b := f.DataTracker.NewTemplate()
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
	// GREG: add streaming create.	f.ApiGroup.POST("/templates/:uuid", createTemplate)
	f.ApiGroup.GET("/templates/:id",
		func(c *gin.Context) {
			res, ok := f.DataTracker.FetchOne(f.DataTracker.NewTemplate(), c.Param(`id`))
			if ok {
				c.JSON(http.StatusOK, backend.AsTemplate(res))
			} else {
				c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusNotFound,
					fmt.Sprintf("templates: Not found: %v", c.Param(`id`))))
			}
		})
	f.ApiGroup.PATCH("/templates/:id",
		func(c *gin.Context) {
			//			updateThing(c, &Template{ID: c.Param(`id`)}, &Template{})
		})
	f.ApiGroup.PUT("/templates/:id",
		func(c *gin.Context) {
			b := f.DataTracker.NewTemplate()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
			}
			if b.ID != c.Param(`id`) {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest,
					fmt.Sprintf("templates: Can not change id: %v -> %v", c.Param(`id`), b.ID)))
			}
			nb, err := f.DataTracker.Update(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err) // GREG: Code
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})
	f.ApiGroup.DELETE("/templates/:id",
		func(c *gin.Context) {
			b := f.DataTracker.NewTemplate()
			b.ID = c.Param(`id`)
			_, err := f.DataTracker.Remove(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err) // GREG: Code
			} else {
				c.JSON(http.StatusNoContent, nil)
			}
		})
}

/*
func BootenvPatch(params templates.PatchBootenvParams, p *models.Principal) middleware.Responder {
	newThing := NewBootenv(params.ID)
	patch, _ := json.Marshal(params.Body)
	item, err := patchThing(newThing, patch)
	if err != nil {
		if err.Code == http.StatusNotFound {
			return templates.NewPatchBootenvNotFound().WithPayload(err)
		}
		if err.Code == http.StatusConflict {
			return templates.NewPatchBootenvConflict().WithPayload(err)
		}
		return templates.NewPatchBootenvExpectationFailed().WithPayload(err)
	}
	original, ok := item.(models.BootenvOutput)
	if !ok {
		e := NewError(http.StatusInternalServerError, "Could not marshal bootenv")
		return templates.NewPatchBootenvInternalServerError().WithPayload(e)
	}
	return templates.NewPatchBootenvOK().WithPayload(&original)
}
*/
