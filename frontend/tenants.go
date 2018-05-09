package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// TenantResponse returned on a successful GET, PUT, PATCH, or POST of a single tenant
// swagger:response
type TenantResponse struct {
	// in: body
	Body *models.Tenant
}

// TenantsResponse returned on a successful GET of all the tenants
// swagger:response
type TenantsResponse struct {
	//in: body
	Body []*models.Tenant
}

// TenantBodyParameter used to inject a Tenant
// swagger:parameters createTenant putTenant
type TenantBodyParameter struct {
	// in: body
	// required: true
	Body *models.Tenant
}

// TenantPatchBodyParameter used to patch a Tenant
// swagger:parameters patchTenant
type TenantPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// TenantPathParameter used to name a Tenant in the path
// swagger:parameters putTenants getTenant putTenant patchTenant deleteTenant headTenant
type TenantPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// TenantListPathParameter used to limit lists of Tenant by path options
// swagger:parameters listTenants listStatsTenants
type TenantListPathParameter struct {
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

// TenantActionsPathParameter used to find a Tenant / Actions in the path
// swagger:parameters getTenantActions
type TenantActionsPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
	// in: query
	Plugin string `json:"plugin"`
}

// TenantActionPathParameter used to find a Tenant / Action in the path
// swagger:parameters getTenantAction
type TenantActionPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
	// in: path
	// required: true
	Cmd string `json:"cmd"`
	// in: query
	Plugin string `json:"plugin"`
}

// TenantActionBodyParameter used to post a Tenant / Action in the path
// swagger:parameters postTenantAction
type TenantActionBodyParameter struct {
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

func (f *Frontend) InitTenantApi() {
	// swagger:route GET /tenants Tenants listTenants
	//
	// Lists Tenants filtered by some parameters.
	//
	// This will show all Tenants by default.
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
	//    200: TenantsResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/tenants",
		func(c *gin.Context) {
			f.List(c, &backend.Tenant{})
		})

	// swagger:route HEAD /tenants Tenants listStatsTenants
	//
	// Stats of the List Tenants filtered by some parameters.
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
	f.ApiGroup.HEAD("/tenants",
		func(c *gin.Context) {
			f.ListStats(c, &backend.Tenant{})
		})

	// swagger:route POST /tenants Tenants createTenant
	//
	// Create a Tenant
	//
	// Create a Tenant from the provided object
	//
	//     Responses:
	//       201: TenantResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/tenants",
		func(c *gin.Context) {
			b := &backend.Tenant{}
			f.Create(c, b)
		})
	// swagger:route GET /tenants/{name} Tenants getTenant
	//
	// Get a Tenant
	//
	// Get the Tenant specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: TenantResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/tenants/:name",
		func(c *gin.Context) {
			f.Fetch(c, &backend.Tenant{}, c.Param(`name`))
		})

	// swagger:route HEAD /tenants/{name} Tenants headTenant
	//
	// See if a Tenant exists
	//
	// Return 200 if the Tenant specifiec by {name} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/tenants/:name",
		func(c *gin.Context) {
			f.Exists(c, &backend.Tenant{}, c.Param(`name`))
		})

	// swagger:route PATCH /tenants/{name} Tenants patchTenant
	//
	// Patch a Tenant
	//
	// Update a Tenant specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: TenantResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/tenants/:name",
		func(c *gin.Context) {
			f.Patch(c, &backend.Tenant{}, c.Param(`name`))
		})

	// swagger:route PUT /tenants/{name} Tenants putTenant
	//
	// Put a Tenant
	//
	// Update a Tenant specified by {name} using a JSON Tenant
	//
	//     Responses:
	//       200: TenantResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/tenants/:name",
		func(c *gin.Context) {
			f.Update(c, &backend.Tenant{}, c.Param(`name`))
		})

	// swagger:route DELETE /tenants/{name} Tenants deleteTenant
	//
	// Delete a Tenant
	//
	// Delete a Tenant specified by {name}
	//
	//     Responses:
	//       200: TenantResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/tenants/:name",
		func(c *gin.Context) {
			f.Remove(c, &backend.Tenant{}, c.Param(`name`))
		})

	tenant := &backend.Tenant{}
	pActions, pAction, pRun := f.makeActionEndpoints(tenant.Prefix(), tenant, "name")

	// swagger:route GET /tenants/{name}/actions Tenants getTenantActions
	//
	// List tenant actions Tenant
	//
	// List Tenant actions for a Tenant specified by {name}
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionsResponse
	//       401: NoTenantResponse
	//       403: NoTenantResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/tenants/:name/actions", pActions)

	// swagger:route GET /tenants/{name}/actions/{cmd} Tenants getTenantAction
	//
	// List specific action for a tenant Tenant
	//
	// List specific {cmd} action for a Tenant specified by {name}
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionResponse
	//       400: ErrorResponse
	//       401: NoTenantResponse
	//       403: NoTenantResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/tenants/:name/actions/:cmd", pAction)

	// swagger:route POST /tenants/{name}/actions/{cmd} Tenants postTenantAction
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
	//       401: NoTenantResponse
	//       403: NoTenantResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/tenants/:name/actions/:cmd", pRun)
}
