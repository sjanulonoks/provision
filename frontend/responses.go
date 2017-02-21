package frontend

import "github.com/rackn/rocket-skates/backend"

// BootEnvResponse returned on a successful GET, PUT, PATCH, or POST of a single bootenv
// swagger:response
type BootEnvResponse struct {
	// in: body
	Body *backend.BootEnv
}

// BootEnvsResponse returned on a successful GET of all the bootenvs
// swagger:response
type BootEnvsResponse struct {
	//in: body
	Body []*backend.BootEnv
}

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

// MachineResponse return on a successful GET, PUT, PATCH or POST of a single Machine
// swagger:response
type MachineResponse struct {
	//in: body
	Body *backend.Machine
}

// MachinesResponse return on a successful GET of all Machines
// swagger:response
type MachinesResponse struct {
	//in: body
	Body []*backend.Machine
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

// TemplateResponse return on a successful GET, PUT, PATCH or POST of a single Template
// swagger:response
type TemplateResponse struct {
	//in: body
	Body *backend.Template
}

// TemplatesResponse return on a successful GET of all templates
// swagger:response
type TemplatesResponse struct {
	//in: body
	Body []*backend.Template
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
