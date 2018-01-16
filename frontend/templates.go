package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// TemplateResponse return on a successful GET, PUT, PATCH or POST of a single Template
// swagger:response
type TemplateResponse struct {
	//in: body
	Body *models.Template
}

// TemplatesResponse return on a successful GET of all templates
// swagger:response
type TemplatesResponse struct {
	//in: body
	Body []*models.Template
}

// TemplateBodyParameter used to inject a Template
// swagger:parameters createTemplate putTemplate
type TemplateBodyParameter struct {
	// in: body
	// required: true
	Body *models.Template
}

// TemplatePatchBodyParameter used to patch a Template
// swagger:parameters patchTemplate
type TemplatePatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// TemplatePathParameter used to id a Template in the path
// swagger:parameters putTemplates getTemplate putTemplate patchTemplate deleteTemplate headTemplate
type TemplatePathParameter struct {
	// in: path
	// required: true
	Id string `json:"id"`
}

// TemplateListPathParameter used to limit lists of Template by path options
// swagger:parameters listTemplates listStatsTemplates
type TemplateListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Available string
	// in: query
	Valid string
	// in: query
	ReadOnly string
	// in: query
	ID string
}

// TemplateActionsPathParameter used to find a Template / Actions in the path
// swagger:parameters getTemplateActions
type TemplateActionsPathParameter struct {
	// in: path
	// required: true
	Id string `json:"id"`
	// in: query
	Plugin string `json:"plugin"`
}

// TemplateActionPathParameter used to find a Template / Action in the path
// swagger:parameters getTemplateAction
type TemplateActionPathParameter struct {
	// in: path
	// required: true
	Id string `json:"id"`
	// in: path
	// required: true
	Cmd string `json:"cmd"`
	// in: query
	Plugin string `json:"plugin"`
}

// TemplateActionBodyParameter used to post a Template / Action in the path
// swagger:parameters postTemplateAction
type TemplateActionBodyParameter struct {
	// in: path
	// required: true
	Id string `json:"id"`
	// in: path
	// required: true
	Cmd string `json:"cmd"`
	// in: query
	Plugin string `json:"plugin"`
	// in: body
	// required: true
	Body map[string]interface{}
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
	//    Available = boolean
	//    Valid = boolean
	//    ReadOnly = boolean
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
			f.List(c, &backend.Template{})
		})

	// swagger:route HEAD /templates Templates listStatsTemplates
	//
	// Stats of the List Templates filtered by some parameters.
	//
	// This will return headers with the stats of the list.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    ID = string
	//    Available = boolean
	//    Valid = boolean
	//    ReadOnly = boolean
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
	//    200: NoContentResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.HEAD("/templates",
		func(c *gin.Context) {
			f.ListStats(c, &backend.Template{})
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
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/templates",
		func(c *gin.Context) {
			b := &backend.Template{}
			f.Create(c, b)
		})

	// swagger:route GET /templates/{id} Templates getTemplate
	//
	// Get a Template
	//
	// Get the Template specified by {id} or return NotFound.
	//
	//     Responses:
	//       200: TemplateResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/templates/:id",
		func(c *gin.Context) {
			f.Fetch(c, &backend.Template{}, c.Param(`id`))
		})

	// swagger:route HEAD /templates/{id} Templates headTemplate
	//
	// See if a Template exists
	//
	// Return 200 if the Template specifiec by {id} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/templates/:id",
		func(c *gin.Context) {
			f.Exists(c, &backend.Template{}, c.Param(`id`))
		})

	// swagger:route PATCH /templates/{id} Templates patchTemplate
	//
	// Patch a Template
	//
	// Update a Template specified by {id} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: TemplateResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/templates/:id",
		func(c *gin.Context) {
			f.Patch(c, &backend.Template{}, c.Param(`id`))
		})

	// swagger:route PUT /templates/{id} Templates putTemplate
	//
	// Put a Template
	//
	// Update a Template specified by {id} using a JSON Template
	//
	//     Responses:
	//       200: TemplateResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/templates/:id",
		func(c *gin.Context) {
			f.Update(c, &backend.Template{}, c.Param(`id`))
		})

	// swagger:route DELETE /templates/{id} Templates deleteTemplate
	//
	// Delete a Template
	//
	// Delete a Template specified by {id}
	//
	//     Responses:
	//       200: TemplateResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/templates/:id",
		func(c *gin.Context) {
			f.Remove(c, &backend.Template{}, c.Param(`id`))
		})

	pActions, pAction, pRun := f.makeActionEndpoints(&backend.Template{}, "id")

	// swagger:route GET /templates/{id}/actions Templates getTemplateActions
	//
	// List template actions Template
	//
	// List Template actions for a Template specified by {id}
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionsResponse
	//       401: NoTemplateResponse
	//       403: NoTemplateResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/templates/:id/actions", pActions)

	// swagger:route GET /templates/{id}/actions/{cmd} Templates getTemplateAction
	//
	// List specific action for a template Template
	//
	// List specific {cmd} action for a Template specified by {id}
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionResponse
	//       400: ErrorResponse
	//       401: NoTemplateResponse
	//       403: NoTemplateResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/templates/:id/actions/:cmd", pAction)

	// swagger:route POST /templates/{id}/actions/{cmd} Templates postTemplateAction
	//
	// Call an action on the node.
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//
	//     Responses:
	//       400: ErrorResponse
	//       200: ActionPostResponse
	//       401: NoTemplateResponse
	//       403: NoTemplateResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/templates/:id/actions/:cmd", pRun)
}
