package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/gin-gonic/gin"
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

// TemplateListPathParameter used to limit lists of Template by path options
// swagger:parameters listTemplates
type TemplateListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	ID string
}

func (f *Frontend) InitTemplateApi() {
	// swagger:route GET /templates Templates listTemplates
	//
	// Lists Templates filtered by some parameters.
	//
	// This will show all Templates by default.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    ID = string
	//
	// Functions:
	//    Eq(value) = Return items that are equal to value
	//    Lt(value) = Return items that are less than value
	//    Lte(value) = Return items that less than or equal to value
	//    Gt(value) = Return items that are greater than value
	//    Gte(value) = Return items that greater than or equal to value
	//    Between(lower,upper) = Return items that are inclusively between lower and upper
	//    Except(lower,upper) = Return items that are not inclusively between lower and upper
	//
	// Example:
	//    ID=fred - returns items named fred
	//    ID=Lt(fred) - returns items that alphabetically less than fred.
	//
	// Responses:
	//    200: TemplatesResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
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
			f.Create(c, b, nil)
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
	//       406: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/templates/:id",
		func(c *gin.Context) {
			f.Patch(c, f.dt.NewTemplate(), c.Param(`id`), nil)
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
			f.Update(c, f.dt.NewTemplate(), c.Param(`id`), nil)
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
	//       409: ErrorResponse
	f.ApiGroup.DELETE("/templates/:id",
		func(c *gin.Context) {
			b := f.dt.NewTemplate()
			b.ID = c.Param(`id`)
			f.Remove(c, b, nil)
		})
}
