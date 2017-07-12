package frontend

import (
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/digitalrebar/go/common/store"
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
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Uuid = UUID string
	//    Name = string
	//    BootEnv = string
	//    Address = IP Address
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
	//    200: MachinesResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
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
			var res store.KeySaver
			var err error
			func() {
				d, unlocker := f.dt.LockEnts(store.KeySaver(b).(Lockable).Locks("create")...)
				defer unlocker()
				_, err = f.dt.Create(d, b)
			}()
			if err != nil {
				be, ok := err.(*backend.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				s, ok := store.KeySaver(b).(Sanitizable)
				if ok {
					res = s.Sanitize()
				} else {
					res = b
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
			b := f.dt.NewMachine()
			var ref store.KeySaver
			func() {
				d, unlocker := f.dt.LockEnts(store.KeySaver(b).(Lockable).Locks("get")...)
				defer unlocker()
				ref = d("machines").Find(uuid)
			}()
			if ref == nil {
				err := &backend.Error{
					Code:  http.StatusNotFound,
					Type:  "API_ERROR",
					Model: "machines",
					Key:   uuid,
				}
				err.Errorf("%s GET Params: %s: Not Found", err.Model, err.Key)
				c.JSON(err.Code, err)
				return
			}
			if !assureAuth(c, f.Logger, ref.Prefix(), "get", ref.Key()) {
				return
			}
			p := backend.AsMachine(ref).GetParams()
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
			b := f.dt.NewMachine()
			var ref store.KeySaver
			func() {
				d, unlocker := f.dt.LockEnts(store.KeySaver(b).(Lockable).Locks("get")...)
				defer unlocker()
				ref = d("machines").Find(uuid)
			}()
			if ref == nil {
				err := &backend.Error{
					Code:  http.StatusNotFound,
					Type:  "API_ERROR",
					Model: "machines",
					Key:   uuid,
				}
				err.Errorf("%s SET Params: %s: Not Found", err.Model, err.Key)
				c.JSON(err.Code, err)
				return
			}
			if !assureAuth(c, f.Logger, ref.Prefix(), "get", ref.Key()) {
				return
			}

			m := backend.AsMachine(ref)
			var err error
			func() {
				d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("update")...)
				defer unlocker()
				err = m.SetParams(d, val)
			}()
			if err != nil {
				be, _ := err.(*backend.Error)
				c.JSON(be.Code, be)
			} else {
				c.JSON(http.StatusOK, val)
			}
		})

}
