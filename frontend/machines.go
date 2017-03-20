package frontend

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

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

// MachineBodyParameter used to inject a Machine
// swagger:parameters createMachine putMachine
type MachineBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Machine
}

// MachinePatchBodyParameter used to patch a Machine
// swagger:parameters patchMachine
type MachinePatchBodyParameter struct {
	// in: body
	// required: true
	Body []JSONPatchOperation
}

// MachinePathParameter used to name a Machine in the path
// swagger:parameters putMachines getMachine putMachine patchMachine deleteMachine
type MachinePathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

func (f *Frontend) InitMachineApi() {
	// swagger:route GET /machines Machines listMachines
	//
	// Lists Machines filtered by some parameters.
	//
	// This will show all Machines by default.
	//
	//     Responses:
	//       200: MachinesResponse
	//       401: ErrorResponse
	f.ApiGroup.GET("/machines",
		func(c *gin.Context) {
			f.List(c, f.dt.NewMachine())
		})

	// swagger:route POST /machines Machines createMachine
	//
	// Create a Machine
	//
	// Create a Machine from the provided object
	//
	//     Responses:
	//       201: MachineResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/machines",
		func(c *gin.Context) {
			b := f.dt.NewMachine()
			f.Create(c, b)
		})
	// swagger:route GET /machines/{name} Machines getMachine
	//
	// Get a Machine
	//
	// Get the Machine specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: MachineResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/machines/:name",
		func(c *gin.Context) {
			f.Fetch(c, f.dt.NewMachine(), c.Param(`name`))
		})

	// swagger:route PATCH /machines/{name} Machines patchMachine
	//
	// Patch a Machine
	//
	// Update a Machine specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: MachineResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/machines/:name",
		func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, backend.NewError("API_ERROR", http.StatusNotImplemented, "machine patch: NOT IMPLEMENTED"))
		})

	// swagger:route PUT /machines/{name} Machines putMachine
	//
	// Put a Machine
	//
	// Update a Machine specified by {name} using a JSON Machine
	//
	//     Responses:
	//       200: MachineResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/machines/:name",
		func(c *gin.Context) {
			f.Update(c, f.dt.NewMachine(), c.Param(`name`))
		})

	// swagger:route DELETE /machines/{name} Machines deleteMachine
	//
	// Delete a Machine
	//
	// Delete a Machine specified by {name}
	//
	//     Responses:
	//       200: MachineResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/machines/:name",
		func(c *gin.Context) {
			b := f.dt.NewMachine()
			b.Name = c.Param(`name`)
			f.Remove(c, b)
		})
}
