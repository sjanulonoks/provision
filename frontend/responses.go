package frontend

import "github.com/rackn/rocket-skates/backend"

// LeaseResponse return on a successful GET, PUT, PATCH or POST of a single Lease
// swagger:response
type LeaseResponse struct {
	//in: body
	Body *backend.Lease
}

// LeasesResponse return on a successful GET of all leases
// swagger:response
type LeasesResponse struct {
	//in: body
	Body []*backend.Lease
}

// ReservationResponse return on a successful GET, PUT, PATCH or POST of a single Reservation
// swagger:response
type ReservationResponse struct {
	//in: body
	Body *backend.Reservation
}

// ReservationsResponse return on a successful GET of all Reservations
// swagger:response
type ReservationsResponse struct {
	//in: body
	Body []*backend.Reservation
}

// SubnetResponse return on a successful GET, PUT, PATCH or POST of a single Subnet
// swagger:response
type SubnetResponse struct {
	//in: body
	Body *backend.Subnet
}

// SubnetsResponse return on a successful GET of all Subnets
// swagger:response
type SubnetsResponse struct {
	//in: body
	Body []*backend.Subnet
}

// UserResponse return on a successful GET, PUT, PATCH or POST of a single User
// swagger:response
type UserResponse struct {
	//in: body
	Body *backend.User
}

// UsersResponse return on a successful GET of all leases
// swagger:response
type UsersResponse struct {
	//in: body
	Body []*backend.User
}

// ErrorResponse is returned whenever an error occurs
// swagger:response
type ErrorResponse struct {
	//in: body
	Body backend.Error
}
