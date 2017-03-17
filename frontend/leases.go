package frontend

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

// LeaseResponse returned on a successful GET, PUT, PATCH, or POST of a single lease
// swagger:response
type LeaseResponse struct {
	// in: body
	Body *backend.Lease
}

// LeasesResponse returned on a successful GET of all the leases
// swagger:response
type LeasesResponse struct {
	//in: body
	Body []*backend.Lease
}

// LeaseBodyParameter used to inject a Lease
// swagger:parameters createLease putLease
type LeaseBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Lease
}

// LeasePatchBodyParameter used to patch a Lease
// swagger:parameters patchLease
type LeasePatchBodyParameter struct {
	// in: body
	// required: true
	Body []JSONPatchOperation
}

// LeasePathParameter used to address a Lease in the path
// swagger:parameters putLeases getLease putLease patchLease deleteLease
type LeasePathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt ipv4
	Address string `json:"address"`
}

func (f *Frontend) InitLeaseApi() {
	// swagger:route GET /leases Leases listLeases
	//
	// Lists Leases filtered by some parameters.
	//
	// This will show all Leases by default.
	//
	//     Responses:
	//       200: LeasesResponse
	//       401: ErrorResponse
	f.ApiGroup.GET("/leases",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, backend.AsLeases(f.dt.FetchAll(f.dt.NewLease())))
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
	//       401: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/leases",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			b := f.dt.NewLease()
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

	// swagger:route GET /leases/{address} Leases getLease
	//
	// Get a Lease
	//
	// Get the Lease specified by {address} or return NotFound.
	//
	//     Responses:
	//       200: LeaseResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/leases/:address",
		func(c *gin.Context) {
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease get: address not valid: %v", c.Param(`address`))))
				return
			}
			res, ok := f.dt.FetchOne(f.dt.NewLease(), backend.Hexaddr(ip))
			if ok {
				c.JSON(http.StatusOK, backend.AsLease(res))
			} else {
				c.JSON(http.StatusNotFound,
					backend.NewError("API_ERROR", http.StatusNotFound,
						fmt.Sprintf("lease get: error not found: %v", c.Param(`address`))))
			}
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
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/leases/:address",
		func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, backend.NewError("API_ERROR", http.StatusNotImplemented, "lease patch: NOT IMPLEMENTED"))
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
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/leases/:address",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease put: address not valid: %v", c.Param(`address`))))
				return
			}
			b := f.dt.NewLease()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				return
			}
			if !b.Addr.Equal(ip) {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease put: error can not change address: %v %v", ip, b.Addr)))
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

	// swagger:route DELETE /leases/{address} Leases deleteLease
	//
	// Delete a Lease
	//
	// Delete a Lease specified by {address}
	//
	//     Responses:
	//       200: LeaseResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/leases/:address",
		func(c *gin.Context) {
			b := f.dt.NewLease()
			b.Addr = net.ParseIP(c.Param(`address`))
			if b.Addr == nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("lease delete: address not valid: %v", c.Param(`address`))))
				return
			}
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
