package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

// SubnetResponse returned on a successful GET, PUT, PATCH, or POST of a single subnet
// swagger:response
type SubnetResponse struct {
	// in: body
	Body *backend.Subnet
}

// SubnetsResponse returned on a successful GET of all the subnets
// swagger:response
type SubnetsResponse struct {
	//in: body
	Body []*backend.Subnet
}

// SubnetBodyParameter used to inject a Subnet
// swagger:parameters createSubnet putSubnet
type SubnetBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Subnet
}

// SubnetPatchBodyParameter used to patch a Subnet
// swagger:parameters patchSubnet
type SubnetPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// SubnetPathParameter used to name a Subnet in the path
// swagger:parameters putSubnets getSubnet putSubnet patchSubnet deleteSubnet
type SubnetPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

func (f *Frontend) InitSubnetApi() {
	// swagger:route GET /subnets Subnets listSubnets
	//
	// Lists Subnets filtered by some parameters.
	//
	// This will show all Subnets by default.
	//
	//     Responses:
	//       200: SubnetsResponse
	//       401: ErrorResponse
	f.ApiGroup.GET("/subnets",
		func(c *gin.Context) {
			f.List(c, f.dt.NewSubnet())
		})

	// swagger:route POST /subnets Subnets createSubnet
	//
	// Create a Subnet
	//
	// Create a Subnet from the provided object
	//
	//     Responses:
	//       201: SubnetResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/subnets",
		func(c *gin.Context) {
			b := f.dt.NewSubnet()
			f.Create(c, b)
		})

	// swagger:route GET /subnets/{name} Subnets getSubnet
	//
	// Get a Subnet
	//
	// Get the Subnet specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: SubnetResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/subnets/:name",
		func(c *gin.Context) {
			f.Fetch(c, f.dt.NewSubnet(), c.Param(`name`))
		})

	// swagger:route PATCH /subnets/{name} Subnets patchSubnet
	//
	// Patch a Subnet
	//
	// Update a Subnet specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: SubnetResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/subnets/:name",
		func(c *gin.Context) {
			f.Patch(c, f.dt.NewSubnet(), c.Param(`name`))
		})

	// swagger:route PUT /subnets/{name} Subnets putSubnet
	//
	// Put a Subnet
	//
	// Update a Subnet specified by {name} using a JSON Subnet
	//
	//     Responses:
	//       200: SubnetResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/subnets/:name",
		func(c *gin.Context) {
			f.Update(c, f.dt.NewSubnet(), c.Param(`name`))
		})

	// swagger:route DELETE /subnets/{name} Subnets deleteSubnet
	//
	// Delete a Subnet
	//
	// Delete a Subnet specified by {name}
	//
	//     Responses:
	//       200: SubnetResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/subnets/:name",
		func(c *gin.Context) {
			b := f.dt.NewSubnet()
			b.Name = c.Param(`name`)
			f.Remove(c, b)
		})
}
