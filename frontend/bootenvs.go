package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// BootEnvResponse returned on a successful GET, PUT, PATCH, or POST of a single bootenv
// swagger:response
type BootEnvResponse struct {
	// in: body
	Body *models.BootEnv
}

// BootEnvsResponse returned on a successful GET of all the bootenvs
// swagger:response
type BootEnvsResponse struct {
	//in: body
	Body []*models.BootEnv
}

// BootEnvBodyParameter used to inject a BootEnv
// swagger:parameters createBootEnv putBootEnv
type BootEnvBodyParameter struct {
	// in: body
	// required: true
	Body *models.BootEnv
}

// BootEnvPatchBodyParameter used to patch a BootEnv
// swagger:parameters patchBootEnv
type BootEnvPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// BootEnvPathParameter used to name a BootEnv in the path
// swagger:parameters putBootEnvs getBootEnv putBootEnv patchBootEnv deleteBootEnv headBootEnv
type BootEnvPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// BootEnvListPathParameter used to limit lists of BootEnv by path options
// swagger:parameters listBootEnvs listStatsBootEnvs
type BootEnvListPathParameter struct {
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
	OnlyUnknown string
	// in: query
	Name string
}

func (f *Frontend) InitBootEnvApi() {
	// swagger:route GET /bootenvs BootEnvs listBootEnvs
	//
	// Lists BootEnvs filtered by some parameters.
	//
	// This will show all BootEnvs by default.
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
	//    OnlyUnknown = boolean
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
	//    Name=Lt(fred)&Available=true - returns items with Name less than fred and Available is true
	//
	// Responses:
	//    200: BootEnvsResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/bootenvs",
		func(c *gin.Context) {
			f.List(c, &backend.BootEnv{})
		})

	// swagger:route HEAD /bootenvs BootEnvs listStatsBootEnvs
	//
	// Stats of the List BootEnvs filtered by some parameters.
	//
	// This will return headers with the stats of the list.
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
	//    OnlyUnknown = boolean
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
	//    Name=Lt(fred)&Available=true - returns items with Name less than fred and Available is true
	//
	// Responses:
	//    200: NoContentResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.HEAD("/bootenvs",
		func(c *gin.Context) {
			f.ListStats(c, &backend.BootEnv{})
		})

	// swagger:route POST /bootenvs BootEnvs createBootEnv
	//
	// Create a BootEnv
	//
	// Create a BootEnv from the provided object
	//
	//     Responses:
	//       201: BootEnvResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/bootenvs",
		func(c *gin.Context) {
			f.Create(c, &backend.BootEnv{})
		})
	// swagger:route GET /bootenvs/{name} BootEnvs getBootEnv
	//
	// Get a BootEnv
	//
	// Get the BootEnv specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: BootEnvResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/bootenvs/:name",
		func(c *gin.Context) {
			f.Fetch(c, &backend.BootEnv{}, c.Param(`name`))
		})

	// swagger:route HEAD /bootenvs/{name} BootEnvs headBootEnv
	//
	// See if a BootEnv exists
	//
	// Return 200 if the BootEnv specifiec by {name} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/bootenvs/:name",
		func(c *gin.Context) {
			f.Exists(c, &backend.BootEnv{}, c.Param(`name`))
		})

	// swagger:route PATCH /bootenvs/{name} BootEnvs patchBootEnv
	//
	// Patch a BootEnv
	//
	// Update a BootEnv specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/bootenvs/:name",
		func(c *gin.Context) {
			f.Patch(c, &backend.BootEnv{}, c.Param(`name`))
		})

	// swagger:route PUT /bootenvs/{name} BootEnvs putBootEnv
	//
	// Put a BootEnv
	//
	// Update a BootEnv specified by {name} using a JSON BootEnv
	//
	//     Responses:
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/bootenvs/:name",
		func(c *gin.Context) {
			f.Update(c, &backend.BootEnv{}, c.Param(`name`))
		})

	// swagger:route DELETE /bootenvs/{name} BootEnvs deleteBootEnv
	//
	// Delete a BootEnv
	//
	// Delete a BootEnv specified by {name}
	//
	//     Responses:
	//       200: BootEnvResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/bootenvs/:name",
		func(c *gin.Context) {
			f.Remove(c, &backend.BootEnv{}, c.Param(`name`))
		})
}
