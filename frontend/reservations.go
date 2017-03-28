package frontend

import (
	"fmt"
	"net"
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

// ReservationResponse returned on a successful GET, PUT, PATCH, or POST of a single reservation
// swagger:response
type ReservationResponse struct {
	// in: body
	Body *backend.Reservation
}

// ReservationsResponse returned on a successful GET of all the reservations
// swagger:response
type ReservationsResponse struct {
	//in: body
	Body []*backend.Reservation
}

// ReservationBodyParameter used to inject a Reservation
// swagger:parameters createReservation putReservation
type ReservationBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Reservation
}

// ReservationPatchBodyParameter used to patch a Reservation
// swagger:parameters patchReservation
type ReservationPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// ReservationPathParameter used to address a Reservation in the path
// swagger:parameters putReservations getReservation putReservation patchReservation deleteReservation
type ReservationPathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt ipv4
	Address string `json:"address"`
}

func (f *Frontend) InitReservationApi() {
	// swagger:route GET /reservations Reservations listReservations
	//
	// Lists Reservations filtered by some parameters.
	//
	// This will show all Reservations by default.
	//
	//     Responses:
	//       200: ReservationsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	f.ApiGroup.GET("/reservations",
		func(c *gin.Context) {
			f.List(c, f.dt.NewReservation())
		})

	// swagger:route POST /reservations Reservations createReservation
	//
	// Create a Reservation
	//
	// Create a Reservation from the provided object
	//
	//     Responses:
	//       201: ReservationResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/reservations",
		func(c *gin.Context) {
			b := f.dt.NewReservation()
			f.Create(c, b)
		})

	// swagger:route GET /reservations/{address} Reservations getReservation
	//
	// Get a Reservation
	//
	// Get the Reservation specified by {address} or return NotFound.
	//
	//     Responses:
	//       200: ReservationResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/reservations/:address",
		func(c *gin.Context) {
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("reservation get: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Fetch(c, f.dt.NewReservation(), backend.Hexaddr(ip))
		})

	// swagger:route PATCH /reservations/{address} Reservations patchReservation
	//
	// Patch a Reservation
	//
	// Update a Reservation specified by {address} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: ReservationResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/reservations/:address",
		func(c *gin.Context) {
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("reservation get: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Patch(c, f.dt.NewReservation(), backend.Hexaddr(ip))
		})

	// swagger:route PUT /reservations/{address} Reservations putReservation
	//
	// Put a Reservation
	//
	// Update a Reservation specified by {address} using a JSON Reservation
	//
	//     Responses:
	//       200: ReservationResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/reservations/:address",
		func(c *gin.Context) {
			ip := net.ParseIP(c.Param(`address`))
			if ip == nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("reservation put: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Update(c, f.dt.NewReservation(), backend.Hexaddr(ip))
		})

	// swagger:route DELETE /reservations/{address} Reservations deleteReservation
	//
	// Delete a Reservation
	//
	// Delete a Reservation specified by {address}
	//
	//     Responses:
	//       200: ReservationResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/reservations/:address",
		func(c *gin.Context) {
			b := f.dt.NewReservation()
			b.Addr = net.ParseIP(c.Param(`address`))
			if b.Addr == nil {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("reservation delete: address not valid: %v", c.Param(`address`))))
				return
			}
			f.Remove(c, b)
		})
}
