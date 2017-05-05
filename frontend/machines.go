package frontend

import (
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/gin-gonic/gin"
	"github.com/pborman/uuid"
)

// MachineResponse return on a successful GET, PUT, PATCH or POST of a single Machine
// swagger:response
type MachineResponse struct {
	// in: body
	Body *backend.Machine
}

// MachinesResponse return on a successful GET of all Machines
// swagger:response
type MachinesResponse struct {
	// in: body
	Body []*backend.Machine
}

// MachineParamsResponse return on a successful GET of all Machine's Params
// swagger:response
type MachineParamsResponse struct {
	// in: body
	Body map[string]interface{}
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
	Body jsonpatch2.Patch
}

// MachinePathParameter used to find a Machine in the path
// swagger:parameters putMachines getMachine putMachine patchMachine deleteMachine getMachineParams postMachineParams
type MachinePathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
}

// MachineParamsBodyParameter used to set Machine Params
// swagger:parameters postMachineParams
type MachineParamsBodyParameter struct {
	// in: body
	// required: true
	Body map[string]interface{}
}

// MachineListPathParameter used to limit lists of Machine by path options
// swagger:parameters listMachines
type MachineListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Uuid string
	// in: query
	Name string
	// in: query
	BootEnv string
	// in: query
	Address string
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       406: ErrorResponse
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/machines",
		func(c *gin.Context) {
			// We don't use f.Create() because we need to be able to assign random
			// UUIDs to new Machines without forcing the client to do so, yet allow them
			// for testing purposes amd if they alrady have a UUID scheme for machines.
			b := f.dt.NewMachine()
			if !assureDecode(c, b) {
				return
			}
			if b.Uuid == nil || len(b.Uuid) == 0 {
				b.Uuid = uuid.NewRandom()
			}
			res, err := f.dt.Create(b)
			if err != nil {
				be, ok := err.(*backend.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				s, ok := res.(Sanitizable)
				if ok {
					s.Sanitize()
				}
				c.JSON(http.StatusCreated, res)
			}
		})

	// swagger:route GET /machines/{uuid} Machines getMachine
	//
	// Get a Machine
	//
	// Get the Machine specified by {uuid} or return NotFound.
	//
	//     Responses:
	//       200: MachineResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/machines/:uuid",
		func(c *gin.Context) {
			f.Fetch(c, f.dt.NewMachine(), c.Param(`uuid`))
		})

	// swagger:route PATCH /machines/{uuid} Machines patchMachine
	//
	// Patch a Machine
	//
	// Update a Machine specified by {uuid} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: MachineResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/machines/:uuid",
		func(c *gin.Context) {
			f.Patch(c, f.dt.NewMachine(), c.Param(`uuid`))
		})

	// swagger:route PUT /machines/{uuid} Machines putMachine
	//
	// Put a Machine
	//
	// Update a Machine specified by {uuid} using a JSON Machine
	//
	//     Responses:
	//       200: MachineResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/machines/:uuid",
		func(c *gin.Context) {
			f.Update(c, f.dt.NewMachine(), c.Param(`uuid`))
		})

	// swagger:route DELETE /machines/{uuid} Machines deleteMachine
	//
	// Delete a Machine
	//
	// Delete a Machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/machines/:uuid",
		func(c *gin.Context) {
			b := f.dt.NewMachine()
			b.Uuid = uuid.Parse(c.Param(`uuid`))
			f.Remove(c, b)
		})

	// swagger:route GET /machines/{uuid}/params Machines getMachineParams
	//
	// List machine params Machine
	//
	// List Machine parms for a Machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineParamsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/machines/:uuid/params",
		func(c *gin.Context) {
			uuid := c.Param(`uuid`)
			ref := f.dt.NewMachine()
			if !assureAuth(c, f.Logger, ref.Prefix(), "get", uuid) {
				return
			}
			res, ok := f.dt.FetchOne(ref, uuid)
			if !ok {
				err := &backend.Error{
					Code:  http.StatusNotFound,
					Type:  "API_ERROR",
					Model: ref.Prefix(),
					Key:   uuid,
				}
				err.Errorf("%s GET Params: %s: Not Found", err.Model, err.Key)
				c.JSON(err.Code, err)
				return
			}
			p := backend.AsMachine(res).GetParams()
			c.JSON(http.StatusOK, p)
		})

	// swagger:route POST /machines/{uuid}/params Machines postMachineParams
	//
	// Set/Replace all the Parameters for a machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineParamsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/machines/:uuid/params",
		func(c *gin.Context) {
			var val map[string]interface{}
			if !assureDecode(c, &val) {
				return
			}
			uuid := c.Param(`uuid`)
			ref := f.dt.NewMachine()
			if !assureAuth(c, f.Logger, ref.Prefix(), "get", uuid) {
				return
			}
			res, ok := f.dt.FetchOne(ref, uuid)
			if !ok {
				err := &backend.Error{
					Code:  http.StatusNotFound,
					Type:  "API_ERROR",
					Model: ref.Prefix(),
					Key:   uuid,
				}
				err.Errorf("%s SET Params: %s: Not Found", err.Model, err.Key)
				c.JSON(err.Code, err)
				return
			}
			m := backend.AsMachine(res)

			err := m.SetParams(val)
			if err != nil {
				be, _ := err.(*backend.Error)
				c.JSON(be.Code, be)
			} else {
				c.JSON(http.StatusOK, val)
			}
		})

}
