package frontend

import (
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
// swagger:parameters putLeases getLease putLease patchLease deleteLease headLease
type LeasePathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt ipv4
	Address string `json:"address"`
}

// LeaseListPathParameter used to limit lists of Lease by path options
// swagger:parameters listLeases listStatsLeases
type LeaseListPathParameter struct {
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

	// swagger:route HEAD /leases Leases listStatsLeases
	//
	// Stats of the List Leases filtered by some parameters.
	//
	// This return headers with the stats of the list.
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
	//    Name=Lt(fred)&Available=true - returns items with Name less than fred and Available is true
	//
	// Responses:
	//    200: NoContentResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.HEAD("/leases",
		func(c *gin.Context) {
			f.ListStats(c, &backend.Lease{})
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
			f.Fetch(c, &backend.Lease{}, ifIpConvertToHex(c.Param(`address`)))
		})

	// swagger:route HEAD /leases/{address} Leases headLease
	//
	// See if a Lease exists
	//
	// Return 200 if the Lease specifiec by {address} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/leases/:address",
		func(c *gin.Context) {
			f.Exists(c, &backend.Lease{}, ifIpConvertToHex(c.Param(`address`)))
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
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/leases/:address",
		func(c *gin.Context) {
			f.Remove(c, &backend.Lease{}, ifIpConvertToHex(c.Param(`address`)))
		})
}
