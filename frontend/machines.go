package frontend

import (
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
	"github.com/pborman/uuid"
)

// MachineResponse return on a successful GET, PUT, PATCH or POST of a single Machine
// swagger:response
type MachineResponse struct {
	// in: body
	Body *models.Machine
}

// MachinesResponse return on a successful GET of all Machines
// swagger:response
type MachinesResponse struct {
	// in: body
	Body []*models.Machine
}

// MachineActionResponse return on a successful GET of a single Machine Action
// swagger:response
type MachineActionResponse struct {
	// in: body
	Body *models.AvailableAction
}

// MachineActionsResponse return on a successful GET of all Machine Actions
// swagger:response
type MachineActionsResponse struct {
	// in: body
	Body []*models.AvailableAction
}

// MachineParamsResponse return on a successful GET of all Machine's Params
// swagger:response
type MachineParamsResponse struct {
	// in: body
	Body map[string]interface{}
}

// MachineActionPostResponse return on a successful POST of action
// swagger:response
type MachineActionPostResponse struct {
	// in: body
	Body string
}

// MachineBodyParameter used to inject a Machine
// swagger:parameters createMachine putMachine
type MachineBodyParameter struct {
	// in: query
	Force string `json:"force"`
	// in: body
	// required: true
	Body *models.Machine
}

// MachinePatchBodyParameter used to patch a Machine
// swagger:parameters patchMachine
type MachinePatchBodyParameter struct {
	// in: query
	Force string `json:"force"`
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// MachinePathParameter used to find a Machine in the path
// swagger:parameters putMachines getMachine putMachine patchMachine deleteMachine postMachineParams getMachineActions
type MachinePathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
}

// MachineGetParamsPathParameter used to find a Machine in the path
// swagger:parameters getMachineParams
type MachineGetParamsPathParameter struct {
	// in: query
	Aggregate string `json:"aggregate"`
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
}

// MachineActionPathParameter used to find a Machine / Action in the path
// swagger:parameters getMachineAction
type MachineActionPathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
	// in: path
	// required: true
	Name string `json:"name"`
}

// MachineActionBodyParameter used to post a Machine / Action in the path
// swagger:parameters postMachineAction
type MachineActionBodyParameter struct {
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
	// in: path
	// required: true
	Name string `json:"name"`
	// in: body
	// required: true
	Body map[string]interface{}
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
	Available string
	// in: query
	Valid string
	// in: query
	Uuid string
	// in: query
	Name string
	// in: query
	BootEnv string
	// in: query
	Address string
	// in: query
	Runnable string
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
	//    Runnable = true/false
	//    Available = boolean
	//    Valid = boolean
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
			f.List(c, &backend.Machine{})
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
			b := &backend.Machine{}
			if !assureDecode(c, b) {
				return
			}
			if b.Uuid == nil || len(b.Uuid) == 0 {
				b.Uuid = uuid.NewRandom()
			}
			var res models.Model
			var err error
			func() {
				d, unlocker := f.dt.LockEnts(models.Model(b).(Lockable).Locks("create")...)
				defer unlocker()
				_, err = f.dt.Create(d, b)
			}()
			if err != nil {
				be, ok := err.(*models.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, models.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				s, ok := models.Model(b).(Sanitizable)
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
			f.Fetch(c, &backend.Machine{}, c.Param(`uuid`))
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
			machine := &backend.Machine{}
			backend.Fill(machine)
			if c.Query("force") == "true" {
				machine.ForceChange()
			}
			f.Patch(c, machine, c.Param(`uuid`))
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
			machine := &backend.Machine{}
			backend.Fill(machine)
			if c.Query("force") == "true" {
				machine.ForceChange()
			}
			f.Update(c, machine, c.Param(`uuid`))
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
			f.Remove(c, &backend.Machine{}, c.Param(`uuid`))
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
			b := &backend.Machine{}
			var ref models.Model

			aggregate := false
			if c.Query("aggregate") == "true" {
				aggregate = true
			}

			p := func() map[string]interface{} {
				d, unlocker := f.dt.LockEnts(models.Model(b).(Lockable).Locks("get")...)
				defer unlocker()
				ref = d("machines").Find(uuid)
				if ref != nil {
					return backend.AsMachine(ref).GetParams(d, aggregate)
				}
				return nil
			}()
			if ref == nil {
				err := &models.Error{
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
			b := &backend.Machine{}
			var ref models.Model
			func() {
				d, unlocker := f.dt.LockEnts(models.Model(b).(Lockable).Locks("get")...)
				defer unlocker()
				ref = d("machines").Find(uuid)
			}()
			if ref == nil {
				err := &models.Error{
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
				be, _ := err.(*models.Error)
				c.JSON(be.Code, be)
			} else {
				c.JSON(http.StatusOK, val)
			}
		})

	// swagger:route GET /machines/{uuid}/actions Machines getMachineActions
	//
	// List machine actions Machine
	//
	// List Machine actions for a Machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineActionsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/machines/:uuid/actions",
		func(c *gin.Context) {
			if !assureAuth(c, f.Logger, "machines", "actions", c.Param(`uuid`)) {
				return
			}
			uuid := c.Param(`uuid`)
			b := &backend.Machine{}
			var ref models.Model
			list := make([]*models.AvailableAction, 0, 0)
			bad := func() bool {
				d, unlocker := f.dt.LockEnts(models.Model(b).(Lockable).Locks("actions")...)
				defer unlocker()
				ref = d("machines").Find(uuid)
				if ref == nil {
					err := &models.Error{
						Code:  http.StatusNotFound,
						Type:  "API_ERROR",
						Model: "machines",
						Key:   uuid,
					}
					err.Errorf("%s Actions Get: %s: Not Found", err.Model, err.Key)
					c.JSON(err.Code, err)
					return true
				}

				m := backend.AsMachine(ref)
				for _, aa := range f.pc.MachineActions.List() {
					if _, err := validateMachineAction(f, d, aa.Command, m, make(map[string]interface{}, 0)); err == nil {
						list = append(list, aa)
					}
				}
				return false
			}()
			if bad {
				return
			}

			c.JSON(http.StatusOK, list)
		})

	// swagger:route GET /machines/{uuid}/actions/{name} Machines getMachineAction
	//
	// List specific action for a machine Machine
	//
	// List specific {name} action for a Machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineActionResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/machines/:uuid/actions/:name",
		func(c *gin.Context) {
			if !assureAuth(c, f.Logger, "machines", c.Param(`name`), c.Param(`uuid`)) {
				return
			}
			uuid := c.Param(`uuid`)
			b := &backend.Machine{}
			var ref models.Model
			var aa *models.AvailableAction
			bad := func() bool {
				d, unlocker := f.dt.LockEnts(models.Model(b).(Lockable).Locks("actions")...)
				defer unlocker()
				ref = d("machines").Find(uuid)
				if ref == nil {
					err := &models.Error{
						Code:  http.StatusNotFound,
						Type:  "API_ERROR",
						Model: "machines",
						Key:   uuid,
					}
					err.Errorf("%s Action Get: %s: Not Found", err.Model, err.Key)
					c.JSON(err.Code, err)
					return true
				}
				m := backend.AsMachine(ref)
				var err *models.Error
				aa, err = validateMachineAction(f, d, c.Param(`name`), m, make(map[string]interface{}, 0))
				if err != nil {
					c.JSON(err.Code, err)
					return true
				}
				return false
			}()

			if bad {
				return
			}

			c.JSON(http.StatusOK, aa)
		})

	// swagger:route POST /machines/{uuid}/actions/{name} Machines postMachineAction
	//
	// Call an action on the node.
	//
	//     Responses:
	//       400: ErrorResponse
	//       200: MachineActionPostResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/machines/:uuid/actions/:name",
		func(c *gin.Context) {
			var val map[string]interface{}
			if !assureDecode(c, &val) {
				return
			}
			uuid := c.Param(`uuid`)
			name := c.Param(`name`)

			ma := &models.MachineAction{Command: name, Params: val}

			b := &backend.Machine{}
			var ref models.Model
			bad := func() bool {
				d, unlocker := f.dt.LockEnts(models.Model(b).(Lockable).Locks("actions")...)
				defer unlocker()
				ref = d("machines").Find(uuid)
				if ref == nil {
					err := &models.Error{
						Code:  http.StatusNotFound,
						Type:  "API_ERROR",
						Model: "machines",
						Key:   uuid,
					}
					err.Errorf("%s Call Action: machine %s: Not Found", err.Model, err.Key)
					c.JSON(err.Code, err)
					return true
				}
				if !assureAuth(c, f.Logger, ref.Prefix(), name, ref.Key()) {
					return true
				}

				m := backend.AsMachine(ref)

				ma.Name = m.Name
				ma.Uuid = m.Uuid
				ma.Address = m.Address
				ma.BootEnv = m.BootEnv

				var err *models.Error
				_, err = validateMachineAction(f, d, name, m, val)
				if err != nil {
					c.JSON(err.Code, err)
					return true
				}
				return false
			}()

			if bad {
				return
			}

			f.pubs.Publish("machines", name, uuid, ma)
			err := f.pc.MachineActions.Run(ma)
			if err != nil {
				be, ok := err.(*models.Error)
				if !ok {
					c.JSON(409, err)
				} else {
					c.JSON(be.Code, be)
				}
			} else {
				c.JSON(http.StatusOK, "")
			}
		})

}

func validateMachineAction(f *Frontend, d backend.Stores, name string, m *backend.Machine, val map[string]interface{}) (*models.AvailableAction, *models.Error) {
	err := &models.Error{
		Code:  http.StatusBadRequest,
		Type:  "API_ERROR",
		Model: "machines",
		Key:   m.Uuid.String(),
	}

	aa, ok := f.pc.MachineActions.Get(name)
	if !ok {
		err.Errorf("%s Call Action: action %s: Not Found", err.Model, name)
		return nil, err
	}

	for _, param := range aa.RequiredParams {
		var obj interface{} = nil
		obj, ok := val[param]
		if !ok {
			obj, ok = m.GetParam(d, param, true)
			if !ok {
				if o := d("profiles").Find(f.dt.GlobalProfileName); o != nil {
					p := backend.AsProfile(o)
					if tobj, ok := p.Params[param]; ok {
						obj = tobj
					}
				}
			}

			// Put into place
			if obj != nil {
				val[param] = obj
			}
		}
		if obj == nil {
			err.Errorf("%s Call Action: machine %s: Missing Parameter %s", err.Model, err.Key, param)
		} else {
			pobj := d("params").Find(param)
			if pobj != nil {
				rp := backend.AsParam(pobj)

				if ev := rp.ValidateValue(obj); ev != nil {
					err.Errorf("%s Call Action machine %s: Invalid Parameter: %s: %s", err.Model, err.Key, param, ev.Error())
				}
			}
		}
	}
	for _, param := range aa.OptionalParams {
		var obj interface{} = nil
		obj, ok := val[param]
		if !ok {
			obj, ok = m.GetParam(d, param, true)
			if !ok {
				if o := d("profiles").Find(f.dt.GlobalProfileName); o != nil {
					p := backend.AsProfile(o)
					if tobj, ok := p.Params[param]; ok {
						obj = tobj
					}
				}
			}

			// Put into place
			if obj != nil {
				val[param] = obj
			}
		}
		if obj != nil {
			pobj := d("params").Find(param)
			if pobj != nil {
				rp := backend.AsParam(pobj)

				if ev := rp.ValidateValue(obj); ev != nil {
					err.Errorf("%s Call Action machine %s: Invalid Parameter: %s: %s", err.Model, err.Key, param, ev.Error())
				}
			}
		}
	}

	if err.HasError() == nil {
		return aa, nil
	}
	return aa, err
}
