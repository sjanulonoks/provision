package frontend

import (
	"fmt"
	"net/http"

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
// swagger:parameters createSubnets putSubnet
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
	Body []JSONPatchOperation
}

// SubnetPathParameter used to name a Subnet in the path
// swagger:parameters putSubnets getSubnet putSubnet patchSubnet deleteSubnet
type SubnetPathParameter struct {
	// in: path
	// required: true
	Name string
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
			c.JSON(http.StatusOK, backend.AsSubnets(f.dt.FetchAll(f.dt.NewSubnet())))
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
	f.ApiGroup.POST("/subnets",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			b := f.dt.NewSubnet()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
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
			res, ok := f.dt.FetchOne(f.dt.NewSubnet(), c.Param(`name`))
			if ok {
				c.JSON(http.StatusOK, backend.AsSubnet(res))
			} else {
				c.JSON(http.StatusNotFound,
					backend.NewError("API_ERROR", http.StatusNotFound,
						fmt.Sprintf("subnet get: error not found: %v", c.Param(`name`))))
			}
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
	f.ApiGroup.PATCH("/subnets/:name",
		func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, backend.NewError("API_ERROR", http.StatusNotImplemented, "subnet patch: NOT IMPLEMENTED"))
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
	f.ApiGroup.PUT("/subnets/:name",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			b := f.dt.NewSubnet()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				return
			}
			if b.Name != c.Param(`name`) {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("subnet put: error can not change name: %v %v", c.Param(`name`), b.Name)))
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
