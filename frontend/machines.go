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

// MachineParamResponse return on a successful GET of a single Machine param
// swagger:response
type MachineParamResponse struct {
	// in: body
	Body interface{}
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

// MachinePatchBodyParameter used to patch a Machine
// swagger:parameters patchMachineParams
type MachinePatchParamsParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

//MachinePostParamParameter used to POST a machine parameter
//swagger:parameters postMachineParam
type MachinePostParamParameter struct {
	// in: body
	// required: true
	Body interface{}
}

//MachinePostParamsParameter used to POST machine parameters
//swagger:parameters postMachineParams
type MachinePostParamsParameter struct {
	// in: body
	// required: true
	Body map[string]interface{}
}

// MachinePathParameter used to find a Machine in the path
// swagger:parameters putMachines getMachine putMachine patchMachine deleteMachine getMachineActions headMachine patchMachineParams postMachineParams
type MachinePathParameter struct {
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
}

// MachinePostParamPathParemeter used to get a single Parameter for a single Machine
// swagger:parameters postMachineParam
type MachinePostParamPathParemeter struct {
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
	// in: path
	//required: true
	Key string `json:"key"`
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

//  MachineGetParamPathParemeter used to get a single Parameter for a single Machine
// swagger:parameters getMachineParam
type MachineGetParamPathParemeter struct {
	// in: query
	Aggregate string `json:"aggregate"`
	// in: path
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID `json:"uuid"`
	// in: path
	//required: true
	Key string `json:"key"`
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

// MachineListPathParameter used to limit lists of Machine by path options
// swagger:parameters listMachines listStatsMachines
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
	ReadOnly string
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
	//    200: MachinesResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/machines",
		func(c *gin.Context) {
			f.List(c, &backend.Machine{})
		})

	// swagger:route HEAD /machines Machines listStatsMachines
	//
	// Stats of the List Machines filtered by some parameters.
	//
	// This will return headers with the stats of the list.
	//
	//   X-DRP-LIST-COUNT - number of objects in the list.
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
	f.ApiGroup.HEAD("/machines",
		func(c *gin.Context) {
			f.ListStats(c, &backend.Machine{})
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
	//       409: ErrorResponse
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
			rt := f.rt(c, b.Locks("create")...)
			rt.Do(func(d backend.Stores) {
				_, err = rt.Create(b)
			})
			if err != nil {
				be, ok := err.(*models.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, models.NewError(c.Request.Method, http.StatusBadRequest, err.Error()))
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

	// swagger:route HEAD /machines/{uuid} Machines headMachine
	//
	// See if a Machine exists
	//
	// Return 200 if the Machine specifiec by {uuid} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/machines/:uuid",
		func(c *gin.Context) {
			f.Exists(c, &backend.Machine{}, c.Param(`uuid`))
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
	//       409: ErrorResponse
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
	//       409: ErrorResponse
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
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/machines/:uuid",
		func(c *gin.Context) {
			f.Remove(c, &backend.Machine{}, c.Param(`uuid`))
		})

	pGetAll, pGetOne, pPatch, pSetThem, pSetOne, pDeleteOne := f.makeParamEndpoints(&backend.Machine{}, "uuid")

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
	f.ApiGroup.GET("/machines/:uuid/params", pGetAll)

	// swagger:route GET /machines/{uuid}/params/{key} Machines getMachineParam
	//
	// Get a single machine parameter
	//
	// Get a single parameter {key} for a Machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineParamResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/machines/:uuid/params/*key", pGetOne)

	// swagger:route DELETE /machines/{uuid}/params/{key} Machines getMachineParam
	//
	// Delete a single machine parameter
	//
	// Delete a single parameter {key} for a Machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineParamResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/machines/:uuid/params/*key", pDeleteOne)

	// swagger:route PATCH /machines/{uuid}/params Machines patchMachineParams
	//
	// Update params for Machine {uuid} with the passed-in patch
	//
	//     Responses:
	//       200: MachineParamsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.PATCH("/machines/:uuid/params", pPatch)

	// swagger:route POST /machines/{uuid}/params Machines postMachineParams
	//
	// Sets parameters for a machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineParamsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/machines/:uuid/params", pSetThem)

	// swagger:route POST /machines/{uuid}/params/{key} Machines postMachineParam
	//
	// Set as single Parameter {key} for a machine specified by {uuid}
	//
	//     Responses:
	//       200: MachineParamResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/machines/:uuid/params/*key", pSetOne)

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
			if !f.assureAuth(c, "machines", "actions", c.Param(`uuid`)) {
				return
			}
			uuid := c.Param(`uuid`)
			b := &backend.Machine{}
			actions := []models.AvailableAction{}
			var err *models.Error
			rt := f.rt(c, b.Locks("actions")...)
			rt.Do(func(d backend.Stores) {
				ref := rt.Find("machines", uuid)
				if ref == nil {
					err = &models.Error{
						Code:  http.StatusNotFound,
						Type:  c.Request.Method,
						Model: "machines",
						Key:   uuid,
					}
					err.Errorf("Not Found")
					return
				}
				m := backend.AsMachine(ref)
				for _, aa := range f.pc.MachineActions.List() {
					if _, err := validateMachineAction(f, rt, aa.Command, m, make(map[string]interface{}, 0)); err == nil {
						actions = append(actions, *aa)
					}
				}
			})
			if err != nil {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, actions)
			}
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
			if !f.assureAuth(c, "machines", c.Param(`name`), c.Param(`uuid`)) {
				return
			}
			uuid := c.Param(`uuid`)
			b := &backend.Machine{}
			rt := f.rt(c, b.Locks("actions")...)
			var action models.AvailableAction
			var err *models.Error
			rt.Do(func(d backend.Stores) {
				ref := rt.Find("machines", uuid)
				if ref == nil {
					err = &models.Error{
						Code:  http.StatusNotFound,
						Type:  c.Request.Method,
						Model: "machines",
						Key:   uuid,
					}
					err.Errorf("Action Get: '%s': Not Found", c.Param(`name`))
					return
				}
				m := backend.AsMachine(ref)
				action, err = validateMachineAction(f, rt, c.Param(`name`), m, make(map[string]interface{}, 0))
			})
			if err != nil {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, action)
			}
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

			if !f.assureAuth(c, "machines", name, uuid) {
				return
			}

			b := &backend.Machine{}
			var ref models.Model
			var err *models.Error
			var ma *models.MachineAction
			rt := f.rt(c, b.Locks("actions")...)
			rt.Do(func(d backend.Stores) {
				ref = rt.Find("machines", uuid)
				if ref == nil {
					err = &models.Error{
						Code:  http.StatusNotFound,
						Type:  "INVOKE",
						Model: "machines",
						Key:   uuid,
					}
					err.Errorf("Not Found")
					return
				}
				res := &models.MachineAction{Command: name, Params: val}
				m := backend.AsMachine(ref)
				res.Name = m.Name
				res.Uuid = m.Uuid
				res.Address = m.Address
				res.BootEnv = m.BootEnv
				if _, err = validateMachineAction(f, rt, name, m, val); err != nil {
					err.Type = "INVOKE"
					return
				}
				ma = res
			})
			if err != nil {
				c.JSON(err.Code, err)
				return
			}
			f.pubs.Publish("machines", name, uuid, ma)
			runErr := f.pc.MachineActions.Run(ma)
			if runErr != nil {
				be, ok := runErr.(*models.Error)
				if !ok {
					c.JSON(409, runErr)
				} else {
					c.JSON(be.Code, be)
				}
			} else {
				c.JSON(http.StatusOK, "")
			}
		})

}

func validateMachineAction(f *Frontend,
	rt *backend.RequestTracker,
	name string,
	m *backend.Machine,
	val map[string]interface{}) (models.AvailableAction, *models.Error) {
	aa := models.AvailableAction{}
	err := &models.Error{
		Code:  http.StatusBadRequest,
		Type:  "GET",
		Model: "machines",
		Key:   m.Key(),
	}

	if raa, ok := f.pc.MachineActions.Get(name); !ok {
		err.Errorf("Action %s: Not Found", name)
		return aa, err
	} else {
		aa = *raa
	}

	for _, param := range aa.RequiredParams {
		var obj interface{} = nil
		obj, ok := val[param]
		if !ok {
			obj, ok = rt.GetParam(m, param, true)
			if !ok {
				if o := rt.Find("profiles", f.dt.GlobalProfileName); o != nil {
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
			err.Errorf("Action %s Missing Parameter %s", name, param)
		} else {
			pobj := rt.Find("params", param)
			if pobj != nil {
				rp := backend.AsParam(pobj)

				if ev := rp.ValidateValue(obj); ev != nil {
					err.Errorf("Action %s: Invalid Parameter: %s: %s", name, param, ev.Error())
				}
			}
		}
	}
	for _, param := range aa.OptionalParams {
		var obj interface{} = nil
		obj, ok := val[param]
		if !ok {
			obj, ok = rt.GetParam(m, param, true)
			if !ok {
				if o := rt.Find("profiles", f.dt.GlobalProfileName); o != nil {
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
			pobj := rt.Find("params", param)
			if pobj != nil {
				rp := backend.AsParam(pobj)

				if ev := rp.ValidateValue(obj); ev != nil {
					err.Errorf("Action %s: Invalid Parameter: %s: %s", name, param, ev.Error())
				}
			}
		}
	}
	if err.HasError() == nil {
		return aa, nil
	}
	return aa, err
}
