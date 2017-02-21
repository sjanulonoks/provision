package frontend

import (
	"net/http"

	"github.com/rackn/rocket-skates/backend"
)

func initTemplateRoutes(dt *backend.DataTracker) {
	mgmtApi.GET(basePath+"/templates",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, dt.AsTemplates(dt.FetchAll(dt.NewTemplate())))
		})
	mgmtApi.POST(basePath+"/templates",
		func(c *gin.Context) {
			b := dt.NewTemplate()
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
	mgmtApi.GET(basePath+"/templates/:name",
		func(c *gin.Context) {
			res, ok := dt.FetchOne(dt.NewTemplate(), c.Param(`name`))
			if ok {
				c.JSON(http.StatusOK, dt.AsTemplate(res))
			} else {
				c.JSON(http.StatusNotFound, err)
			}
		})
	mgmtApi.PATCH(basePath+"/templates/:name",
		func(c *gin.Context) {
			//			updateThing(c, &Template{Name: c.Param(`name`)}, &Template{})
		})
	mgmtApi.PUT(basePath+"/templates/:name",
		func(c *gin.Context) {
			b := dt.NewTemplate()
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
	mgmtApi.DELETE(basePath+"/templates/:name",
		func(c *gin.Context) {
			b := dt.NewTemplate()
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
func BootenvPatch(params templates.PatchBootenvParams, p *models.Principal) middleware.Responder {
	newThing := NewBootenv(params.Name)
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
