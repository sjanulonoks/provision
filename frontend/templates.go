package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
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
// swagger:parameters createTemplate putTemplate
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
	Body jsonpatch2.Patch
}

// TemplatePathParameter used to name a Template in the path
// swagger:parameters putTemplates getTemplate putTemplate patchTemplate deleteTemplate
type TemplatePathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	f.ApiGroup.GET("/templates",
		func(c *gin.Context) {
			f.List(c, f.dt.NewTemplate())
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/templates",
		func(c *gin.Context) {
			b := f.dt.NewTemplate()
			f.Create(c, b)
		})

	// swagger:route GET /templates/{name} Templates getTemplate
	//
	// Get a Template
	//
	// Get the Template specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: TemplateResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/templates/:id",
		func(c *gin.Context) {
			f.Fetch(c, f.dt.NewTemplate(), c.Param(`id`))
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/templates/:id",
		func(c *gin.Context) {
			f.Patch(c, f.dt.NewTemplate(), c.Param(`id`))
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/templates/:id",
		func(c *gin.Context) {
			f.Update(c, f.dt.NewTemplate(), c.Param(`id`))
		})

	// swagger:route DELETE /templates/{name} Templates deleteTemplate
	//
	// Delete a Template
	//
	// Delete a Template specified by {name}
	//
	//     Responses:
	//       200: TemplateResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/templates/:id",
		func(c *gin.Context) {
			b := f.dt.NewTemplate()
			b.ID = c.Param(`id`)
			f.Remove(c, b)
		})
}
