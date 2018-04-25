package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// RoleResponse returned on a successful GET, PUT, PATCH, or POST of a single role
// swagger:response
type RoleResponse struct {
	// in: body
	Body *models.Role
}

// RolesResponse returned on a successful GET of all the roles
// swagger:response
type RolesResponse struct {
	//in: body
	Body []*models.Role
}

// RoleBodyParameter used to inject a Role
// swagger:parameters createRole putRole
type RoleBodyParameter struct {
	// in: body
	// required: true
	Body *models.Role
}

// RolePatchBodyParameter used to patch a Role
// swagger:parameters patchRole
type RolePatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// RolePathParameter used to name a Role in the path
// swagger:parameters putRoles getRole putRole patchRole deleteRole headRole
type RolePathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// RoleListPathParameter used to limit lists of Role by path options
// swagger:parameters listRoles listStatsRoles
type RoleListPathParameter struct {
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

// RoleActionsPathParameter used to find a Role / Actions in the path
// swagger:parameters getRoleActions
type RoleActionsPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
	// in: query
	Plugin string `json:"plugin"`
}

// RoleActionPathParameter used to find a Role / Action in the path
// swagger:parameters getRoleAction
type RoleActionPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
	// in: path
	// required: true
	Cmd string `json:"cmd"`
	// in: query
	Plugin string `json:"plugin"`
}

// RoleActionBodyParameter used to post a Role / Action in the path
// swagger:parameters postRoleAction
type RoleActionBodyParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
	// in: path
	// required: true
	Cmd string `json:"cmd"`
	// in: query
	Plugin string `json:"plugin"`
	// in: body
	// required: true
	Body map[string]interface{}
}

func (f *Frontend) InitRoleApi() {
	// swagger:route GET /roles Roles listRoles
	//
	// Lists Roles filtered by some parameters.
	//
	// This will show all Roles by default.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Name = string
	//    Reboot = boolean
	//    BootEnv = string
	//    Available = boolean
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
	//    200: RolesResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/roles",
		func(c *gin.Context) {
			f.List(c, &backend.Role{})
		})

	// swagger:route HEAD /roles Roles listStatsRoles
	//
	// Stats of the List Roles filtered by some parameters.
	//
	// This will return headers with the stats of the list.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Name = string
	//    Reboot = boolean
	//    BootEnv = string
	//    Available = boolean
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
	f.ApiGroup.HEAD("/roles",
		func(c *gin.Context) {
			f.ListStats(c, &backend.Role{})
		})

	// swagger:route POST /roles Roles createRole
	//
	// Create a Role
	//
	// Create a Role from the provided object
	//
	//     Responses:
	//       201: RoleResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/roles",
		func(c *gin.Context) {
			b := &backend.Role{}
			f.Create(c, b)
		})
	// swagger:route GET /roles/{name} Roles getRole
	//
	// Get a Role
	//
	// Get the Role specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: RoleResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/roles/:name",
		func(c *gin.Context) {
			f.Fetch(c, &backend.Role{}, c.Param(`name`))
		})

	// swagger:route HEAD /roles/{name} Roles headRole
	//
	// See if a Role exists
	//
	// Return 200 if the Role specifiec by {name} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/roles/:name",
		func(c *gin.Context) {
			f.Exists(c, &backend.Role{}, c.Param(`name`))
		})

	// swagger:route PATCH /roles/{name} Roles patchRole
	//
	// Patch a Role
	//
	// Update a Role specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: RoleResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/roles/:name",
		func(c *gin.Context) {
			f.Patch(c, &backend.Role{}, c.Param(`name`))
		})

	// swagger:route PUT /roles/{name} Roles putRole
	//
	// Put a Role
	//
	// Update a Role specified by {name} using a JSON Role
	//
	//     Responses:
	//       200: RoleResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/roles/:name",
		func(c *gin.Context) {
			f.Update(c, &backend.Role{}, c.Param(`name`))
		})

	// swagger:route DELETE /roles/{name} Roles deleteRole
	//
	// Delete a Role
	//
	// Delete a Role specified by {name}
	//
	//     Responses:
	//       200: RoleResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/roles/:name",
		func(c *gin.Context) {
			f.Remove(c, &backend.Role{}, c.Param(`name`))
		})

	role := &backend.Role{}
	pActions, pAction, pRun := f.makeActionEndpoints(role.Prefix(), role, "name")

	// swagger:route GET /roles/{name}/actions Roles getRoleActions
	//
	// List role actions Role
	//
	// List Role actions for a Role specified by {name}
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionsResponse
	//       401: NoRoleResponse
	//       403: NoRoleResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/roles/:name/actions", pActions)

	// swagger:route GET /roles/{name}/actions/{cmd} Roles getRoleAction
	//
	// List specific action for a role Role
	//
	// List specific {cmd} action for a Role specified by {name}
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionResponse
	//       400: ErrorResponse
	//       401: NoRoleResponse
	//       403: NoRoleResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/roles/:name/actions/:cmd", pAction)

	// swagger:route POST /roles/{name}/actions/{cmd} Roles postRoleAction
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
	//       401: NoRoleResponse
	//       403: NoRoleResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/roles/:name/actions/:cmd", pRun)
}
