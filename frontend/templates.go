package frontend

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

// TemplateResponse return on a successful GET, PUT, PATCH or POST of a single Template
// swagger:response
type TemplateResponse struct {
	//in: body
	Body *backend.Template
}

// TemplatesResponse return on a successful GET of all templates
// swagger:response
type TemplatesResponse struct {
	//in: body
	Body []*backend.Template
}

// TemplateBodyParameter used to inject a Template
// swagger:parameters createTemplates putTemplate
type TemplateBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Template
}

// TemplatePatchBodyParameter used to patch a Template
// swagger:parameters patchTemplate
type TemplatePatchBodyParameter struct {
	// in: body
	// required: true
	Body []JSONPatchOperation
}

// TemplatePathParameter used to name a Template in the path
// swagger:parameters putTemplates getTemplate putTemplate patchTemplate deleteTemplate
type TemplatePathParameter struct {
	// in: path
	// required: true
	Name string
}

func (f *Frontend) InitTemplateApi() {
	// swagger:route GET /templates Templates listTemplates
	//
	// Lists Templates filtered by some parameters.
	//
	// This will show all Templates by default.
	//
	//     Responses:
	//       200: TemplatesResponse
	//       401: ErrorResponse
	f.ApiGroup.GET("/templates",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, backend.AsTemplates(f.dt.FetchAll(f.dt.NewTemplate())))
		})

	// swagger:route POST /templates Templates createTemplate
	//
	// Create a Template
	//
	// Create a Template from the provided object
	//
	//     Responses:
	//       201: TemplateResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	f.ApiGroup.POST("/templates",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			b := f.dt.NewTemplate()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				return
			}
			nb, err := f.dt.Create(b)
			if err != nil {
				ne, ok := err.(*backend.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				c.JSON(http.StatusCreated, nb)
			}
		})

	// GREG: add streaming create.	f.ApiGroup.POST("/templates/:uuid", createTemplate)

	// swagger:route GET /templates/{name} Templates getTemplate
	//
	// Get a Template
	//
	// Get the Template specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: TemplateResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/templates/:id",
		func(c *gin.Context) {
			res, ok := f.dt.FetchOne(f.dt.NewTemplate(), c.Param(`id`))
			if ok {
				c.JSON(http.StatusOK, backend.AsTemplate(res))
			} else {
				c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusNotFound,
					fmt.Sprintf("templates get: Not Found: %v", c.Param(`id`))))
			}
		})

	// swagger:route PATCH /templates/{name} Templates patchTemplate
	//
	// Patch a Template
	//
	// Update a Template specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: TemplateResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.PATCH("/templates/:id",
		func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, backend.NewError("API_ERROR", http.StatusNotImplemented, "template patch: NOT IMPLEMENTED"))
		})

	// swagger:route PUT /templates/{name} Templates putTemplate
	//
	// Put a Template
	//
	// Update a Template specified by {name} using a JSON Template
	//
	//     Responses:
	//       200: TemplateResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.PUT("/templates/:id",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			b := f.dt.NewTemplate()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				return
			}
			if b.ID != c.Param(`id`) {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest,
					fmt.Sprintf("templates put: Can not change id: %v -> %v", c.Param(`id`), b.ID)))
				return
			}
			nb, err := f.dt.Update(b)
			if err != nil {
				ne, ok := err.(*backend.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})

	// swagger:route DELETE /templates/{name} Templates deleteTemplate
	//
	// Delete a Template
	//
	// Delete a Template specified by {name}
	//
	//     Responses:
	//       200: TemplateResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/templates/:id",
		func(c *gin.Context) {
			b := f.dt.NewTemplate()
			b.ID = c.Param(`id`)
			nb, err := f.dt.Remove(b)
			if err != nil {
				ne, ok := err.(*backend.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})
}
