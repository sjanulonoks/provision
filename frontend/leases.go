package frontend

import (
	"fmt"
	"net"
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// LeaseResponse returned on a successful GET, PUT, PATCH, or POST of a single lease
// swagger:response
type LeaseResponse struct {
	// in: body
	Body *models.Lease
}

// LeasesResponse returned on a successful GET of all the leases
// swagger:response
type LeasesResponse struct {
	//in: body
	Body []*models.Lease
}

// LeaseBodyParameter used to inject a Lease
// swagger:parameters createLease putLease
type LeaseBodyParameter struct {
	// in: body
	// required: true
	Body *models.Lease
}

// LeasePatchBodyParameter used to patch a Lease
// swagger:parameters patchLease
type LeasePatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// LeasePathParameter used to address a Lease in the path
// swagger:parameters putLeases getLease putLease patchLease deleteLease
type LeasePathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt ipv4
	Address string `json:"address"`
}

// LeaseListPathParameter used to limit lists of Lease by path options
// swagger:parameters listLeases
type LeaseListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Addr string
	// in: query
	Token string
	// in: query
	Strategy string
	// in: query
	ExpireTime string
}

func (f *Frontend) InitLeaseApi() {
	// swagger:route GET /leases Leases listLeases
	//
	// Lists Leases filtered by some parameters.
	//
	// This will show all Leases by default.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Addr = IP Address
	//    Token = string
	//    Strategy = string
	//    ExpireTime = Date/Time
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
	//    200: LeasesResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/leases",
		func(c *gin.Context) {
			f.List(c, &backend.Lease{})
		})

	// swagger:route POST /leases Leases createLease
	//
	// Create a Lease
	//
	// Create a Lease from the provided object
	//
	//     Responses:
	//       201: LeaseResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/leases",
		func(c *gin.Context) {
			b := &backend.Lease{}
			f.Create(c, b)
		})
	// swagger:route GET /leases/{address} Leases getLease
	//
	// Get a Lease
	//
	// Get the Lease specified by {address} or return NotFound.
	//
	//     Responses:
	//       200: LeaseResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/leases/:address",
		func(c *gin.Context) {
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					models.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease get: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Fetch(c, &backend.Lease{}, models.Hexaddr(ip))
		})

	// swagger:route PATCH /leases/{address} Leases patchLease
	//
	// Patch a Lease
	//
	// Update a Lease specified by {address} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: LeaseResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/leases/:address",
		func(c *gin.Context) {
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					models.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease get: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Patch(c, &backend.Lease{}, models.Hexaddr(ip))
		})

	// swagger:route PUT /leases/{address} Leases putLease
	//
	// Put a Lease
	//
	// Update a Lease specified by {address} using a JSON Lease
	//
	//     Responses:
	//       200: LeaseResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/leases/:address",
		func(c *gin.Context) {
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					models.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease put: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Update(c, &backend.Lease{}, models.Hexaddr(ip))
		})

	// swagger:route DELETE /leases/{address} Leases deleteLease
	//
	// Delete a Lease
	//
	// Delete a Lease specified by {address}
	//
	//     Responses:
	//       200: LeaseResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/leases/:address",
		func(c *gin.Context) {
			addr := net.ParseIP(c.Param(`address`))
			if addr == nil {
				c.JSON(http.StatusBadRequest,
					models.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease delete: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Remove(c, &backend.Lease{}, models.Hexaddr(addr))
		})
}
