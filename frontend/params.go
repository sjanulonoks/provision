package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

// ParamResponse returned on a successful GET, PUT, PATCH, or POST of a single param
// swagger:response
type ParamResponse struct {
	// in: body
	Body *backend.Param
}

// ParamsResponse returned on a successful GET of all the params
// swagger:response
type ParamsResponse struct {
	//in: body
	Body []*backend.Param
}

// ParamBodyParameter used to inject a Param
// swagger:parameters createParam putParam
type ParamBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Param
}

// ParamPatchBodyParameter used to patch a Param
// swagger:parameters patchParam
type ParamPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// ParamPathParameter used to name a Param in the path
// swagger:parameters getParam putParam patchParam deleteParam
type ParamPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

func (f *Frontend) InitParamApi() {
	// swagger:route GET /params Params listParams
	//
	// Lists Params filtered by some parameters.
	//
	// This will show all Params by default.
	//
	//     Responses:
	//       200: ParamsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	f.ApiGroup.GET("/params",
		func(c *gin.Context) {
			f.List(c, f.dt.NewParam())
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
			b := f.dt.NewParam()
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
			f.Fetch(c, f.dt.NewParam(), c.Param(`name`))
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
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/params/:name",
		func(c *gin.Context) {
			f.Patch(c, f.dt.NewParam(), c.Param(`name`))
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
			f.Update(c, f.dt.NewParam(), c.Param(`name`))
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
			b := f.dt.NewParam()
			b.Name = c.Param(`name`)
			f.Remove(c, b)
		})
}
