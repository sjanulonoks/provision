package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// ParamResponse returned on a successful GET, PUT, PATCH, or POST of a single param
// swagger:response
type ParamResponse struct {
	// in: body
	Body *models.Param
}

// ParamsResponse returned on a successful GET of all the params
// swagger:response
type ParamsResponse struct {
	//in: body
	Body []*models.Param
}

// ParamParamsResponse return on a successful GET of all Param's Params
// swagger:response
type ParamParamsResponse struct {
	// in: body
	Body map[string]interface{}
}

// ParamBodyParameter used to inject a Param
// swagger:parameters createParam putParam
type ParamBodyParameter struct {
	// in: body
	// required: true
	Body *models.Param
}

// ParamPatchBodyParameter used to patch a Param
// swagger:parameters patchParam
type ParamPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// ParamPathParameter used to name a Param in the path
// swagger:parameters putParams getParam putParam patchParam deleteParam getParamParams postParamParams
type ParamPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// ParamParamsBodyParameter used to set Param Params
// swagger:parameters postParamParams
type ParamParamsBodyParameter struct {
	// in: body
	// required: true
	Body map[string]interface{}
}

// ParamListPathParameter used to limit lists of Param by path options
// swagger:parameters listParams
type ParamListPathParameter struct {
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
	Name string
}

func (f *Frontend) InitParamApi() {
	// swagger:route GET /params Params listParams
	//
	// Lists Params filtered by some parameters.
	//
	// This will show all Params by default.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Name = string
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
	//    Name=fred - returns items named fred
	//    Name=Lt(fred) - returns items that alphabetically less than fred.
	//
	// Responses:
	//    200: ParamsResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/params",
		func(c *gin.Context) {
			f.List(c, &backend.Param{})
		})

	// swagger:route POST /params Params createParam
	//
	// Create a Param
	//
	// Create a Param from the provided object
	//
	//     Responses:
	//       201: ParamResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/params",
		func(c *gin.Context) {
			b := &backend.Param{}
			f.Create(c, b)
		})
	// swagger:route GET /params/{name} Params getParam
	//
	// Get a Param
	//
	// Get the Param specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: ParamResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/params/:name",
		func(c *gin.Context) {
			f.Fetch(c, &backend.Param{}, c.Param(`name`))
		})

	// swagger:route HEAD /params/{name} Params
	//
	// See if a Param exists
	//
	// Return 200 if the Param specifiec by {name} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/params/:name",
		func(c *gin.Context) {
			f.Exists(c, &backend.Param{}, c.Param(`name`))
		})

	// swagger:route PATCH /params/{name} Params patchParam
	//
	// Patch a Param
	//
	// Update a Param specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: ParamResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/params/:name",
		func(c *gin.Context) {
			f.Patch(c, &backend.Param{}, c.Param(`name`))
		})

	// swagger:route PUT /params/{name} Params putParam
	//
	// Put a Param
	//
	// Update a Param specified by {name} using a JSON Param
	//
	//     Responses:
	//       200: ParamResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/params/:name",
		func(c *gin.Context) {
			f.Update(c, &backend.Param{}, c.Param(`name`))
		})

	// swagger:route DELETE /params/{name} Params deleteParam
	//
	// Delete a Param
	//
	// Delete a Param specified by {name}
	//
	//     Responses:
	//       200: ParamResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/params/:name",
		func(c *gin.Context) {
			f.Remove(c, &backend.Param{}, c.Param(`name`))

		})

}
